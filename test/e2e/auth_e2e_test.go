package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authpb "github.com/ParkieV/auth-service/internal/infrastructure/api/grpc"
)

const (
	restURL  = "http://localhost:8080/api"
	grpcAddr = "localhost:9090"
)

func TestE2E_RegisterLoginRefresh(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}

	// 1) Register
	reqBody := `{"email":"e2e@test.com","password":"StrongPass123"}`
	resp, err := client.Post(restURL+"/register", "application/json", strings.NewReader(reqBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var reg struct {
		UserID string `json:"user_id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&reg))
	require.NotEmpty(t, reg.UserID)

	// 2) Login
	resp2, err := client.Post(restURL+"/login", "application/json", strings.NewReader(reqBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp2.StatusCode)
	var login struct {
		JWT          string `json:"jwt"`
		RefreshToken string `json:"refresh_token"`
	}
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&login))
	require.NotEmpty(t, login.JWT)
	require.NotEmpty(t, login.RefreshToken)

	// 3) Refresh via gRPC
	conn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()
	grpcClient := authpb.NewAuthServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rr, err := grpcClient.Refresh(ctx, &authpb.RefreshRequest{
		RefreshToken: login.RefreshToken,
	})
	require.NoError(t, err)
	require.NotEmpty(t, rr.Jwt)
	require.NotEmpty(t, rr.RefreshToken)
}
