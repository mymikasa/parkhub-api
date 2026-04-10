package main

import (
	"fmt"
	"log"
	"net"
	"os"

	identitygrpc "github.com/parkhub/api/internal/domains/identity/grpc"
	"github.com/parkhub/api/internal/domains/identity/repository/dao"
	"github.com/parkhub/api/internal/registry"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "parkhub:parkhub@tcp(localhost:3306)/parkhub_identity?charset=utf8mb4&parseTime=True&loc=Local"
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(
		&dao.Tenant{},
	); err != nil {
		log.Fatalf("failed to auto migrate: %v", err)
	}

	reg := registry.New()
	identitygrpc.RegisterServices(reg, db)

	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50051"
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	reflection.Register(s)
	reg.RegisterAll(s)

	log.Printf("gRPC server listening on :%s", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
