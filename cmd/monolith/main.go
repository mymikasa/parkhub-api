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
	smsdomain "github.com/parkhub/api/internal/domains/sms/domain"
	smsgrpc "github.com/parkhub/api/internal/domains/sms/grpc"
	smsdao "github.com/parkhub/api/internal/domains/sms/repository/dao"
	smsservice "github.com/parkhub/api/internal/domains/sms/service"
	"github.com/parkhub/api/internal/health"
	"github.com/parkhub/api/internal/middleware"
	"github.com/parkhub/api/internal/registry"
	pkgmetrics "github.com/parkhub/api/pkg/metrics"
	"github.com/parkhub/api/pkg/slogutil"
	"github.com/parkhub/api/pkg/telemetry"
	"github.com/redis/go-redis/v9"
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

	pkgmetrics.Init(telemetryProviders.MeterProvider)

	db, err := gorm.Open(mysql.Open(cfg.Database.DSN()), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}

	if err := db.AutoMigrate(&dao.Tenant{}, &dao.User{}, &smsdao.SmsRecord{}); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	reg := registry.New()
	smsSvc := smsgrpc.RegisterServices(reg, db, rdb)
	smsVerifyFn := func(ctx context.Context, phone, code string) error {
		return smsSvc.VerifyCode(ctx, &smsservice.VerifyCodeRequest{
			Phone:   phone,
			Code:    code,
			Purpose: smsdomain.SmsPurposeLogin,
		})
	}
	identitygrpc.RegisterServices(reg, db, rdb, cfg.Auth, smsVerifyFn)

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
			middleware.UnaryLoggingInterceptor(logger),
			middleware.UnaryAuthContextInterceptor(),
			middleware.UnaryRecoveryInterceptor(logger),
		),
	)
	reflection.Register(grpcServer)
	reg.RegisterAll(grpcServer)

	healthServer := health.NewServer()
	health.RegisterGRPC(grpcServer, healthServer)

	ready := true

	metricsMux := http.NewServeMux()
	metricsMux.Handle(cfg.Telemetry.Metrics.Path, telemetryProviders.MetricsHandler)
	metricsHTTP := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.MetricsPort),
		Handler: metricsMux,
	}

	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		if !ready {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	healthHTTP := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.HTTPHealthPort),
		Handler: healthMux,
	}

	errCh := make(chan error, 3)

	go func() {
		slog.Info("metrics server listening",
			slog.Int("port", cfg.Server.MetricsPort),
			slog.String("path", cfg.Telemetry.Metrics.Path),
		)
		if err := metricsHTTP.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("serve metrics: %w", err)
		}
	}()

	go func() {
		slog.Info("health server listening", slog.Int("port", cfg.Server.HTTPHealthPort))
		if err := healthHTTP.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("serve health: %w", err)
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
		ready = false
		healthServer.SetNotServing()
		gracefulStopGRPC(grpcServer)
		shutdownHTTP(metricsHTTP, healthHTTP)
		return err
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}

	ready = false
	healthServer.SetNotServing()
	gracefulStopGRPC(grpcServer)
	shutdownHTTP(metricsHTTP, healthHTTP)

	return nil
}

func shutdownHTTP(servers ...*http.Server) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, s := range servers {
		_ = s.Shutdown(shutdownCtx)
	}
}

func gracefulStopGRPC(grpcServer *grpc.Server) {
	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		slog.Warn("gRPC graceful stop timed out, forcing stop")
		grpcServer.Stop()
	}
}
