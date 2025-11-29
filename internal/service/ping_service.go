package service

import (
	"context"
	"errors"
	"fmt"
	"time"
	"upbot-server-go/internal/models"
	"upbot-server-go/internal/repository"

	"github.com/go-redis/redis/v8"
)

// PingService defines the business logic for pings.
type PingService interface {
	CreatePing(email string, req CreatePingRequest) (*models.Task, error)
}

type pingService struct {
	repo        repository.TaskRepository
	redisClient *redis.Client
}

// NewPingService creates a new instance of PingService.
func NewPingService(repo repository.TaskRepository, redisClient *redis.Client) PingService {
	return &pingService{
		repo:        repo,
		redisClient: redisClient,
	}
}

type CreatePingRequest struct {
	URL     string
	WebHook string
}

func (s *pingService) CreatePing(email string, req CreatePingRequest) (*models.Task, error) {
	// 1. Get User
	user, err := s.repo.GetUserByEmail(email)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// 2. Check Task Limit (Business Logic)
	activeCount, err := s.repo.CountActiveTasksByUserID(user.ID)
	if err != nil {
		return nil, err
	}
	if activeCount >= 5 {
		return nil, errors.New("task limit reached: you can only have 5 active tasks")
	}

	// 3. Check Duplicate (Business Logic)
	existingTask, _ := s.repo.FindByURLAndUserID(req.URL, user.ID)
	if existingTask != nil {
		return nil, errors.New("task already exists for this URL")
	}

	// 4. Prepare Task
	var webHook *string
	notifyDiscord := false
	if req.WebHook != "" {
		webHook = &req.WebHook
		notifyDiscord = true
	}

	newTask := &models.Task{
		URL:           req.URL,
		IsActive:      true,
		WebHook:       webHook,
		NotifyDiscord: notifyDiscord,
		UserID:        user.ID,
	}

	// 5. Save to DB
	if err := s.repo.Create(newTask); err != nil {
		return nil, err
	}

	// 6. Add to Redis Queue
	taskMember := fmt.Sprintf("%d|%s", newTask.ID, newTask.URL)
	err = s.redisClient.ZAdd(context.Background(), "ping_queue", &redis.Z{
		Score:  float64(time.Now().Add(10 * time.Second).Unix()),
		Member: taskMember,
	}).Err()

	if err != nil {
		// Note: In a real system, you might want to rollback the DB creation or have a retry mechanism
		return nil, fmt.Errorf("failed to schedule task: %w", err)
	}

	return newTask, nil
}
