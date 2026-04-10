package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/parkhub/api/internal/config"
	identitygrpc "github.com/parkhub/api/internal/domains/identity/grpc"
	"github.com/parkhub/api/internal/domains/identity/repository/dao"
	"github.com/parkhub/api/internal/middleware"
	"github.com/parkhub/api/internal/registry"
	"github.com/parkhub/api/pkg/slogutil"
	"github.com/parkhub/api/pkg/telemetry"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "parkhub failed to start: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger, err := slogutil.NewLogger(os.Stdout, slogutil.Options{
		Level:  cfg.Telemetry.Log.Level,
		Format: cfg.Telemetry.Log.Format,
	})
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}
	logger = logger.With(
		slog.String("service", cfg.Telemetry.ServiceName),
		slog.String("environment", cfg.Telemetry.Environment),
	)
	slog.SetDefault(logger)

	telemetryProviders, err := telemetry.Init(ctx, telemetry.Config{
		ServiceName:   cfg.Telemetry.ServiceName,
		Environment:   cfg.Telemetry.Environment,
		TraceEndpoint: cfg.Telemetry.Trace.Endpoint,
		TraceInsecure: cfg.Telemetry.Trace.Insecure,
		SamplingRatio: cfg.Telemetry.Trace.SamplingRatio,
	})
	if err != nil {
		return fmt.Errorf("init telemetry: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := telemetryProviders.Shutdown(shutdownCtx); err != nil {
			slog.Error("failed to shutdown telemetry", slog.String("error", err.Error()))
		}
	}()

	db, err := gorm.Open(mysql.Open(cfg.Database.DSN()), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}

	if err := db.AutoMigrate(&dao.Tenant{}); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}

	reg := registry.New()
	identitygrpc.RegisterServices(reg, db)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
	if err != nil {
		return fmt.Errorf("listen gRPC: %w", err)
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler(
			otelgrpc.WithTracerProvider(telemetryProviders.TracerProvider),
			otelgrpc.WithMeterProvider(telemetryProviders.MeterProvider),
		)),
		grpc.ChainUnaryInterceptor(
			middleware.UnaryRecoveryInterceptor(logger),
			middleware.UnaryLoggingInterceptor(logger),
		),
	)
	reflection.Register(grpcServer)
	reg.RegisterAll(grpcServer)

	metricsMux := http.NewServeMux()
	metricsMux.Handle(cfg.Telemetry.Metrics.Path, telemetryProviders.MetricsHandler)
	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.MetricsPort),
		Handler: metricsMux,
	}

	errCh := make(chan error, 2)

	go func() {
		slog.Info("metrics server listening",
			slog.Int("port", cfg.Server.MetricsPort),
			slog.String("path", cfg.Telemetry.Metrics.Path),
		)
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("serve metrics: %w", err)
		}
	}()

	go func() {
		slog.Info("gRPC server listening", slog.Int("port", cfg.Server.GRPCPort))
		if err := grpcServer.Serve(lis); err != nil {
			errCh <- fmt.Errorf("serve gRPC: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		stop()
		grpcServer.GracefulStop()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = metricsServer.Shutdown(shutdownCtx)
		return err
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}

	grpcServer.GracefulStop()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := metricsServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("shutdown metrics server: %w", err)
	}

	return nil
}
