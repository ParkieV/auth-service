package config

import (
	"encoding/base64"
	"log"
	"time"

	"github.com/spf13/viper"
)

type ServerConfig struct {
	RESTPort int `mapstructure:"rest_port"`
	GRPCPort int `mapstructure:"grpc_port"`
}

type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type RedisConfig struct {
	Addr string `mapstructure:"addr"`
	DB   int    `mapstructure:"db"`
}

type RabbitMQConfig struct {
	URL string `mapstructure:"url"`
}

type KeycloakConfig struct {
	URL      string `mapstructure:"url"`
	Realm    string `mapstructure:"realm"`
	ClientID string `mapstructure:"client_id"`
	Secret   string `mapstructure:"secret"`
}

type JWTConfig struct {
	HMACSecret string        `mapstructure:"hmac_secret"`
	AccessTTL  time.Duration `mapstructure:"access_ttl"`
	RefreshTTL time.Duration `mapstructure:"refresh_ttl"`
}

func (j *JWTConfig) HmacKey() []byte {
	hmacKey, err := base64.StdEncoding.DecodeString(j.HMACSecret)
	if err != nil {
		log.Fatal("invalid hmac_secret: must be base64-encoded")
	}
	return hmacKey
}

type EmailConfig struct {
	From            string        `mapstructure:"from"`
	SMTPHost        string        `mapstructure:"smtp_host"`
	SMTPPort        int           `mapstructure:"smtp_port"`
	SMTPUser        string        `mapstructure:"smtp_user"`
	SMTPPass        string        `mapstructure:"smtp_pass"`
	UseTLS          bool          `mapstructure:"use_tls"`
	DialTimeout     time.Duration `mapstructure:"dial_timeout"`
	ConfirmationTTL time.Duration `mapstructure:"confirmation_ttl"`
}

type LogstashConfig struct {
	TCPAddr string `mapstructure:"tcp_addr"`
}

type CryptoParams struct {
	Time    uint32
	Memory  uint32
	Threads uint8
	SaltLen uint32
	KeyLen  uint32
}

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Postgres PostgresConfig `mapstructure:"postgres"`
	Redis    RedisConfig    `mapstructure:"redis"`
	RabbitMQ RabbitMQConfig `mapstructure:"rabbitmq"`
	Keycloak KeycloakConfig `mapstructure:"keycloak"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Email    EmailConfig    `mapstructure:"email"`
	Logstash LogstashConfig `mapstructure:"logstash"`
	Crypto   CryptoParams   `mapstructure:"crypto"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
