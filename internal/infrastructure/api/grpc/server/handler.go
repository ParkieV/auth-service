package server

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/ParkieV/auth-service/internal/domain"
	authpb "github.com/ParkieV/auth-service/internal/infrastructure/api/grpc"
	"github.com/ParkieV/auth-service/internal/usecase"
)

type AuthServer struct {
	authpb.UnimplementedAuthServiceServer
	registerUC *usecase.RegisterUsecase
	loginUC    *usecase.LoginUsecase
	refreshUC  *usecase.RefreshUsecase
	logoutUC   *usecase.LogoutUsecase
	verifyUC   *usecase.VerifyUsecase
}

func NewAuthServer(
	registerUC *usecase.RegisterUsecase,
	loginUC *usecase.LoginUsecase,
	refreshUC *usecase.RefreshUsecase,
	logoutUC *usecase.LogoutUsecase,
	verifyUC *usecase.VerifyUsecase,
) *AuthServer {
	return &AuthServer{
		registerUC: registerUC,
		loginUC:    loginUC,
		refreshUC:  refreshUC,
		logoutUC:   logoutUC,
		verifyUC:   verifyUC,
	}
}

func (s *AuthServer) Register(
	ctx context.Context,
	req *authpb.RegisterRequest,
) (*authpb.RegisterResponse, error) {
	id, err := s.registerUC.Register(ctx, req.Email, req.Password)
	switch {
	case err == nil:
		return &authpb.RegisterResponse{UserId: id}, nil
	case errors.Is(err, domain.ErrInvalidEmail):
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	case errors.Is(err, usecase.ErrEmailExists):
		return nil, status.Errorf(codes.AlreadyExists, err.Error())
	default:
		return nil, status.Errorf(codes.Internal, "internal error")
	}
}

func (s *AuthServer) Login(
	ctx context.Context,
	req *authpb.LoginRequest,
) (*authpb.LoginResponse, error) {
	at, rt, err := s.loginUC.Login(ctx, req.Email, req.Password)
	switch {
	case err == nil:
		return &authpb.LoginResponse{Jwt: at, RefreshToken: rt}, nil
	case errors.Is(err, usecase.ErrNotConfirmed):
		return nil, status.Errorf(codes.FailedPrecondition, err.Error())
	case errors.Is(err, usecase.ErrUserNotFound),
		errors.Is(err, usecase.ErrInvalidCredentials):
		return nil, status.Errorf(codes.Unauthenticated, err.Error())
	default:
		return nil, status.Errorf(codes.Internal, "internal error")
	}
}

func (s *AuthServer) Refresh(
	ctx context.Context,
	req *authpb.RefreshRequest,
) (*authpb.RefreshResponse, error) {
	at, rt, err := s.refreshUC.Refresh(ctx, req.RefreshToken)
	switch {
	case err == nil:
		return &authpb.RefreshResponse{Jwt: at, RefreshToken: rt}, nil
	case errors.Is(err, usecase.ErrInvalidRefreshToken):
		return nil, status.Errorf(codes.Unauthenticated, err.Error())
	default:
		return nil, status.Errorf(codes.Internal, "internal error")
	}
}

func (s *AuthServer) Logout(
	ctx context.Context,
	req *authpb.LogoutRequest,
) (*emptypb.Empty, error) {
	if err := s.logoutUC.Logout(ctx, req.RefreshToken); err != nil {
		return nil, status.Errorf(codes.Internal, "internal error")
	}
	return &emptypb.Empty{}, nil
}

func (s *AuthServer) Verify(
	ctx context.Context,
	req *authpb.VerifyRequest,
) (*authpb.VerifyResponse, error) {
	res, err := s.verifyUC.Verify(ctx, req.Token)
	switch {
	case err == nil && res.Active:
		return &authpb.VerifyResponse{UserId: res.UserID, Active: true}, nil
	case errors.Is(err, usecase.ErrTokenInvalid):
		return nil, status.Errorf(codes.Unauthenticated, err.Error())
	default:
		return nil, status.Errorf(codes.Internal, "internal error")
	}
}
