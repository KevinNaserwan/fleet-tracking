package platform

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/KevinNaserwan/fleet-tracking/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Dependencies struct {
	DB       *pgxpool.Pool
	Redis    *redis.Client
	RabbitMQ *RabbitMQ
}

func Open(ctx context.Context, cfg config.Config) (*Dependencies, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	poolConfig.ConnConfig.RuntimeParams["application_name"] = cfg.ServiceName
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 1
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		Password:     cfg.RedisPassword,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})
	return &Dependencies{
		DB:       pool,
		Redis:    redisClient,
		RabbitMQ: NewRabbitMQ(cfg.RabbitMQURL),
	}, nil
}

func (d *Dependencies) Close() {
	if d == nil {
		return
	}
	if d.RabbitMQ != nil {
		_ = d.RabbitMQ.Close()
	}
	if d.Redis != nil {
		_ = d.Redis.Close()
	}
	if d.DB != nil {
		d.DB.Close()
	}
}

type RabbitMQ struct {
	url string
}

func NewRabbitMQ(url string) *RabbitMQ {
	return &RabbitMQ{url: url}
}

func (r *RabbitMQ) Check(ctx context.Context) error {
	parsed, err := url.Parse(r.url)
	if err != nil || parsed.Hostname() == "" {
		return fmt.Errorf("invalid RabbitMQ URL")
	}
	port := parsed.Port()
	if port == "" {
		switch parsed.Scheme {
		case "amqp":
			port = "5672"
		case "amqps":
			port = "5671"
		default:
			return fmt.Errorf("unsupported RabbitMQ URL scheme %q", parsed.Scheme)
		}
	}
	dialer := net.Dialer{Timeout: 3 * time.Second}
	connection, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(parsed.Hostname(), port))
	if err != nil {
		return err
	}
	return connection.Close()
}

func (r *RabbitMQ) Close() error {
	return nil
}
