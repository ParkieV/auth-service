package server

import (
	"github.com/ParkieV/auth-service/internal/usecase"
	"google.golang.org/grpc"

	authpb "github.com/ParkieV/auth-service/internal/infrastructure/api/grpc"
)

func RegisterGRPC(
	s *grpc.Server,
	registerUC *usecase.RegisterUsecase,
	loginUC *usecase.LoginUsecase,
	refreshUC *usecase.RefreshUsecase,
) {
	authpb.RegisterAuthServiceServer(s, NewAuthServer(registerUC, loginUC, refreshUC))
}
