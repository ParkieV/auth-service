package server

import (
	"context"

	authpb "github.com/ParkieV/auth-service/internal/infrastructure/api/grpc"
	"github.com/ParkieV/auth-service/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthServer struct {
	authpb.UnimplementedAuthServiceServer
	registerUC *usecase.RegisterUsecase
	loginUC    *usecase.LoginUsecase
	refreshUC  *usecase.RefreshUsecase
}

func NewAuthServer(
	registerUC *usecase.RegisterUsecase,
	loginUC *usecase.LoginUsecase,
	refreshUC *usecase.RefreshUsecase,
) *AuthServer {
	return &AuthServer{registerUC: registerUC, loginUC: loginUC, refreshUC: refreshUC}
}

func (s *AuthServer) Register(
	ctx context.Context,
	req *authpb.RegisterRequest,
) (*authpb.RegisterResponse, error) {
	userID, err := s.registerUC.Register(req.Email, req.Password)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	return &authpb.RegisterResponse{UserId: userID}, nil
}

func (s *AuthServer) Login(
	ctx context.Context,
	req *authpb.LoginRequest,
) (*authpb.LoginResponse, error) {
	access, refresh, err := s.loginUC.Login(req.Email, req.Password)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, err.Error())
	}
	return &authpb.LoginResponse{Jwt: access, RefreshToken: refresh}, nil
}

func (s *AuthServer) Refresh(
	ctx context.Context,
	req *authpb.RefreshRequest,
) (*authpb.RefreshResponse, error) {
	access, refresh, err := s.refreshUC.Refresh(req.RefreshToken)
	if err != nil {
		if err == usecase.ErrInvalidRefreshToken {
			return nil, status.Errorf(codes.Unauthenticated, err.Error())
		}
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return &authpb.RefreshResponse{Jwt: access, RefreshToken: refresh}, nil
}
