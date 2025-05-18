FROM golang:1.21-alpine AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o auth-service ./cmd/server

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
WORKDIR /root/
COPY --from=build /app/auth-service .
COPY configs/config.yaml configs/config.yaml
ENTRYPOINT ["./auth-service", "--config", "configs/config.yaml"]