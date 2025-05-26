package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"github.com/ParkieV/auth-service/internal/config"
	authpb "github.com/ParkieV/auth-service/internal/infrastructure/api/grpc"
	"github.com/ParkieV/auth-service/internal/infrastructure/api/grpc/server"
	"github.com/ParkieV/auth-service/internal/infrastructure/api/rest"
	"github.com/ParkieV/auth-service/internal/infrastructure/auth_client"
	"github.com/ParkieV/auth-service/internal/infrastructure/broker"
	"github.com/ParkieV/auth-service/internal/infrastructure/cache"
	"github.com/ParkieV/auth-service/internal/infrastructure/db"
	"github.com/ParkieV/auth-service/internal/infrastructure/email"
	"github.com/ParkieV/auth-service/internal/usecase"
)

func main() {
	cfgPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		slog.Error("cannot load config", "err", err)
		os.Exit(1)
	}

	log := slog.Default()

	pg, err := db.NewPostgres(cfg.Postgres, log)
	if err != nil {
		log.Error("postgres init", "err", err)
		os.Exit(1)
	}

	redisCache := cache.NewRedisCache(cfg.Redis, log)

	mq, err := broker.NewPublisher(cfg.RabbitMQ.URL, log, "auth.events", "send-email")
	if err != nil {
		log.Error("rabbitmq init", "err", err)
		os.Exit(1)
	}

	_ = email.NewSMTPMailer(cfg.Email)

	kc, err := auth_client.NewDBTokenRepository(cfg.Postgres, cfg.JWT, log)
	if err != nil {
		panic(fmt.Sprintf("Cannot connect to database: %s", err))
	}

	registerUC := usecase.NewRegisterUsecase(pg, mq, kc, cfg.Email.ConfirmationTTL, log)
	loginUC := usecase.NewLoginUsecase(pg, kc, redisCache, log)
	refreshUC := usecase.NewRefreshUsecase(kc, redisCache, cfg.JWT.RefreshTTL, log)
	logoutUC := usecase.NewLogoutUsecase(kc, redisCache, log)
	verifyUC := usecase.NewVerifyUsecase(kc, log)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	rest.RegisterHandlers(router, registerUC, loginUC, refreshUC, logoutUC, verifyUC)

	httpSrv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.RESTPort),
		Handler: router,
	}

	go func() {
		log.Info("REST listening", "port", cfg.Server.RESTPort)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("REST server error", "err", err)
			panic(fmt.Sprintf("Cannot load REST server: %s", err))
		}
	}()

	grpcSrv := grpc.NewServer()
	authSrv := server.NewAuthServer(registerUC, loginUC, refreshUC, logoutUC, verifyUC)
	authpb.RegisterAuthServiceServer(grpcSrv, authSrv)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
	if err != nil {
		log.Error("gRPC listen", "err", err)
		os.Exit(1)
	}
	go func() {
		log.Info("gRPC listening", "port", cfg.Server.GRPCPort)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Error("gRPC server error", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Info("shutdown initiated")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = httpSrv.Shutdown(ctx)
	grpcSrv.GracefulStop()
	_ = mq.Close()
	_ = pg.DB().Close()

	log.Info("shutdown complete")
}
