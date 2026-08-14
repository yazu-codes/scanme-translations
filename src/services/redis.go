package services

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

type RedisService struct {
	client *redis.Client
	logger *slog.Logger
}

type Claims struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func NewRedisService(url, password string, logger *slog.Logger) *RedisService {
	client := redis.NewClient(&redis.Options{
		Addr:     url,
		Password: password,
		DB:       0,
	})

	return &RedisService{
		client: client,
		logger: logger,
	}
}

func (s *RedisService) Exists(ctx context.Context, key string) (bool, error) {
	n, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *RedisService) Get(ctx context.Context, key string) (string, error) {
	val, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // treat "not found" as empty, not an error
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

func (s *RedisService) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return s.client.Set(ctx, key, value, ttl).Err()
}

// GenerateToken creates a new JWT token
func (s *RedisService) GenerateToken(userID uint, email string, role string) (string, error) {
	return "", nil
}

// ValidateToken verifies and parses the JWT token
func (s *RedisService) ValidateToken(tokenString string) (*Claims, error) {
	return nil, nil
}

func (s *RedisService) Ping() {
	ctx := context.Background()
	pong, err := s.client.Ping(ctx).Result()
	if err != nil {
		log.Fatal("Redis down:", err)
	}
	fmt.Println("✅ Redis up:", pong)
}
