package integration

import (
	"context"
	"fmt"
	"os"
	"testing"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/ParkieV/auth-service/internal/config"
)

var (
	RedisConfig config.RedisConfig
	PGConfig    config.PostgresConfig
	RedisCont   tc.Container
	PGCont      tc.Container
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Запускаем Redis
	rreq := tc.ContainerRequest{
		Image:        "redis:7",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp"),
	}
	rcont, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{ContainerRequest: rreq, Started: true})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start redis: %v\n", err)
		os.Exit(1)
	}
	RedisCont = rcont
	{
		host, _ := rcont.Host(ctx)
		port, _ := rcont.MappedPort(ctx, "6379")
		RedisConfig = config.RedisConfig{Addr: fmt.Sprintf("%s:%s", host, port.Port()), DB: 0}
	}

	// Запускаем Postgres
	preq := tc.ContainerRequest{
		Image:        "postgres:15",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "auth",
			"POSTGRES_PASSWORD": "secret",
			"POSTGRES_DB":       "authdb",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}
	pcont, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{ContainerRequest: preq, Started: true})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start postgres: %v\n", err)
		RedisCont.Terminate(ctx)
		os.Exit(1)
	}
	PGCont = pcont
	{
		host, _ := pcont.Host(ctx)
		port, _ := pcont.MappedPort(ctx, "5432")
		PGConfig = config.PostgresConfig{
			Host:     host,
			Port:     port.Int(),
			User:     "auth",
			Password: "secret",
			DBName:   "authdb",
			SSLMode:  "disable",
		}
	}

	code := m.Run()

	RedisCont.Terminate(ctx)
	PGCont.Terminate(ctx)
	os.Exit(code)
}
