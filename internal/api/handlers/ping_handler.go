package handlers

import (
	"net/http"
	"upbot-server-go/internal/service"

	"github.com/gin-gonic/gin"
)

type PingHandler struct {
	service service.PingService
}

func NewPingHandler(service service.PingService) *PingHandler {
	return &PingHandler{service: service}
}

type CreatePingRequest struct {
	Url     string `json:"url" binding:"required,url"`
	WebHook string `json:"webHook"`
}

func (h *PingHandler) CreatePing(c *gin.Context) {
	var req CreatePingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// In a real app, get email from context (set by auth middleware)
	// email := c.GetString("email")
	// For now, let's assume a test email or get it from a header for demonstration if auth isn't set up
	email := "test@example.com" 
	if val, exists := c.Get("email"); exists {
		email = val.(string)
	}

	task, err := h.service.CreatePing(email, service.CreatePingRequest{
		URL:     req.Url,
		WebHook: req.WebHook,
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Task created successfully",
		"taskId":  task.ID,
		"url":     task.URL,
	})
}
