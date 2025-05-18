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
	"github.com/ParkieV/auth-service/internal/infrastructure/broker"
	"github.com/ParkieV/auth-service/internal/infrastructure/cache"
	"github.com/ParkieV/auth-service/internal/infrastructure/db"
	"github.com/ParkieV/auth-service/internal/infrastructure/email"
	"github.com/ParkieV/auth-service/internal/infrastructure/keycloak"
	"github.com/ParkieV/auth-service/internal/usecase"
)

func main() {
	cfgPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("cannot load config: %v", err)
	}

	// Infra
	pg, err := db.NewPostgres(cfg.Postgres)
	if err != nil {
		log.Fatalf("postgres init: %v", err)
	}
	cache := cache.NewRedisCache(cfg.Redis)
	mq, err := broker.NewPublisher(cfg.RabbitMQ.URL)
	if err != nil {
		log.Fatalf("rabbitmq init: %v", err)
	}
	mailer := email.NewSMTPMailer(cfg.Email)
	kc := keycloak.NewClient(cfg.Keycloak)

	// Usecases
	registerUC := usecase.NewRegisterUsecase(pg, mq, cfg.JWT.TTL)
	loginUC := usecase.NewLoginUsecase(pg, kc, cache)
	refreshUC := usecase.NewRefreshUsecase(kc, cache, cfg.JWT.TTL)

	// REST
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	rest.RegisterHandlers(router, registerUC, loginUC, refreshUC)
	httpSrv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.RESTPort),
		Handler: router,
	}
	go func() {
		log.Printf("REST listening on %d", cfg.Server.RESTPort)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("REST error: %v", err)
		}
	}()

	// gRPC
	grpcSrv := grpc.NewServer()
	server.RegisterGRPC(grpcSrv, registerUC, loginUC, refreshUC)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
	if err != nil {
		log.Fatalf("gRPC listen: %v", err)
	}
	go func() {
		log.Printf("gRPC listening on %d", cfg.Server.GRPCPort)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Fatalf("gRPC error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	httpSrv.Shutdown(ctx)
	grpcSrv.GracefulStop()
	mq.Close()
	pg.DB().Close()
	log.Println("shutdown complete")
}
