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
	logoutUC *usecase.LogoutUsecase,
	verifyUC *usecase.VerifyUsecase,
) {
	authpb.RegisterAuthServiceServer(s, NewAuthServer(registerUC, loginUC, refreshUC, logoutUC, verifyUC))
}
