package main

import (
	"log"
	"upbot-server-go/config"
	"upbot-server-go/internal/api/handlers"
	"upbot-server-go/internal/infrastructure"
	"upbot-server-go/internal/models"
	"upbot-server-go/internal/repository"
	"upbot-server-go/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Load Config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Infrastructure (DB)
	db, err := infrastructure.NewDatabaseConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto Migrate
	if err := db.AutoMigrate(&models.User{}, &models.Task{}, &models.Log{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// 3. Repository Layer
	taskRepo := repository.NewTaskRepository(db)

	// 4. Service Layer
	pingService := service.NewPingService(taskRepo)

	// 5. Handler Layer
	pingHandler := handlers.NewPingHandler(pingService)

	// 6. Router Setup
	r := gin.Default()

	// Middleware (Mock Auth for now)
	r.Use(func(c *gin.Context) {
		// In reality, verify token and set email
		c.Set("email", "test@example.com")
		c.Next()
	})

	api := r.Group("/api")
	{
		api.POST("/ping", pingHandler.CreatePing)
	}

	// 7. Start Server
	log.Printf("Server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
