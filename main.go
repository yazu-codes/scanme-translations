package main

import (
	"log/slog"
	"os"

	"github.com/redis/go-redis/v9"
	"github.com/yazu-codes/scanme-translations.git/src/api"
	"github.com/yazu-codes/scanme-translations.git/src/api/handlers"
	"github.com/yazu-codes/scanme-translations.git/src/services"
	"github.com/yazu-codes/scanme-translations.git/src/util"
)

func redisClient(url, pass string) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv(url),
		Password: os.Getenv(pass),
	})
	return rdb
}

func main() {
	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}),
	)
	logger = logger.With(slog.String("component", "auth_service"))

	var config *util.ConfigReader = util.NewConfigReader()
	config.Setup()

	server := api.NewServer(config.Server.ConstructUrl(), logger)
	server.SetupDefaultConfig()

	// Initialize services and repositories
	redisService := services.NewRedisService(config.RedisConfig.Url, config.RedisConfig.Password, logger)
	translationService := services.NewTranslationService(config.GoogleCreds)

	translateHandler := handlers.NewTranslateHandler(redisService, translationService, logger)

	server.Router.POST("/translate", translateHandler.Translate)

	server.Run()
}
