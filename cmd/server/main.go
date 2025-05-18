package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"github.com/ParkieV/auth-service/internal/config"
	"github.com/ParkieV/auth-service/internal/infrastructure/api/grpc/server"
	"github.com/ParkieV/auth-service/internal/infrastructure/api/rest"
	"github.com/ParkieV/auth-service/internal/infrastructure/email"
	"github.com/ParkieV/auth-service/internal/infrastructure/keycloak"
)

func main() {
	cfgPath := flag.String("config", "", "path to config file")
	flag.Parse()
	if *cfgPath == "" {
		log.Fatal("config file is required")
	}

	// 1. Load config
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 2. Init infra
	pg, err := postgres.NewPostgres(cfg.Postgres)
	if err != nil {
		log.Fatalf("postgres init: %v", err)
	}
	rdb := redis.NewRedisCache(cfg.Redis)
	mq, err := rabbitmq.NewPublisher(cfg.RabbitMQ.URL)
	if err != nil {
		log.Fatalf("rabbitmq init: %v", err)
	}
	mailer := email.NewSMTPMailer(cfg.Email)
	kc := keycloak.NewClient(cfg.Keycloak)

	// 3. Wire usecase
	registerUC := refresh.NewRegisterUsecase(pg.UserRepository(), mq, mailer, kc, rdb, cfg.JWT)
	loginUC := refresh.NewLoginUsecase(pg.UserRepository(), kc, rdb, cfg.JWT)

	// 4. Start REST
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	rest.RegisterHandlers(router, registerUC, loginUC)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.RESTPort),
		Handler: router,
	}
	go func() {
		log.Printf("REST server listening on %d", cfg.Server.RESTPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("REST server error: %v", err)
		}
	}()

	// 5. Start gRPC
	grpcSrv := grpc.NewServer()
	server.RegisterGRPC(grpcSrv, registerUC, loginUC)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
	if err != nil {
		log.Fatalf("gRPC listen failed: %v", err)
	}
	go func() {
		log.Printf("gRPC server listening on %d", cfg.Server.GRPCPort)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	// 6. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	grpcSrv.GracefulStop()
	log.Println("servers stopped")
}
