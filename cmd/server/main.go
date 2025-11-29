package main

import (
	"log"
	"upbot-server-go/config"
	"upbot-server-go/internal/api/handlers"
	"upbot-server-go/internal/api/middleware"
	"upbot-server-go/internal/infrastructure"
	"upbot-server-go/internal/models"
	"upbot-server-go/internal/repository"
	"upbot-server-go/internal/service"
	"upbot-server-go/internal/worker"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Load Config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Infrastructure
	// Database
	db, err := infrastructure.NewDatabaseConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	// Redis
	redisClient, err := infrastructure.NewRedisClient(cfg.RedisAddr)
	if err != nil {
		log.Fatalf("Failed to connect to redis: %v", err)
	}
	// Email
	emailClient := infrastructure.NewEmailClient(cfg.ResendAPIKey)

	// Auto Migrate
	if err := db.AutoMigrate(&models.User{}, &models.Task{}, &models.Log{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// 3. Repository Layer
	taskRepo := repository.NewTaskRepository(db)
	logRepo := repository.NewLogRepository(db)

	// 4. Service Layer
	pingService := service.NewPingService(taskRepo, redisClient)
	authService := service.NewAuthService(taskRepo, cfg.JWTSecret)

	// 5. Handler Layer
	pingHandler := handlers.NewPingHandler(pingService)
	authHandler := handlers.NewAuthHandler(authService)

	// 6. Workers
	pingWorker := worker.NewPingWorker(redisClient, taskRepo, logRepo, db)
	notiWorker := worker.NewNotificationWorker(redisClient, taskRepo, emailClient)

	go pingWorker.Start()
	go notiWorker.Start()

	// 7. Router Setup
	r := gin.Default()

	// Public Routes
	r.POST("/auth/google", authHandler.GoogleLogin)

	// Protected Routes
	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	{
		api.POST("/ping", pingHandler.CreatePing)
	}

	// 8. Start Server
	log.Printf("Server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
