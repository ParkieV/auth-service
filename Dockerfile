FROM golang:1.23-bookworm AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" \
    -o /bin/server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot AS app
WORKDIR /app

COPY --from=builder /bin/server /app/server
COPY configs /app/configs

EXPOSE 8080 9090
ENTRYPOINT ["/app/server", "-config", "configs/config.yaml"]