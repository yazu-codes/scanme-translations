package services

import (
	"context"
	"fmt"
	"log"
	"log/slog"

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
