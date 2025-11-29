// Package worker handles background jobs
package worker

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"upbot-server-go/internal/models"
	"upbot-server-go/internal/repository"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type PingWorker struct {
	redisClient *redis.Client
	taskRepo    repository.TaskRepository
	logRepo     repository.LogRepository
	db          *gorm.DB
}

func NewPingWorker(redisClient *redis.Client, taskRepo repository.TaskRepository, logRepo repository.LogRepository, db *gorm.DB) *PingWorker {
	return &PingWorker{
		redisClient: redisClient,
		taskRepo:    taskRepo,
		logRepo:     logRepo,
		db:          db,
	}
}

func (w *PingWorker) Start() {
	log.Println("Starting Ping Worker...")
	for {
		w.processBatch()
		time.Sleep(1 * time.Second)
	}
}

func (w *PingWorker) processBatch() {
	ctx := context.Background()
	now := time.Now().Unix()

	tasks, err := w.redisClient.ZRangeByScore(ctx, "ping_queue", &redis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%d", now),
	}).Result()

	if err != nil {
		log.Printf("Error fetching from queue: %v", err)
		return
	}

	if len(tasks) == 0 {
		return
	}

	for _, taskStr := range tasks {
		w.processTask(ctx, taskStr)
	}
}

func (w *PingWorker) processTask(ctx context.Context, taskStr string) {
	parts := strings.SplitN(taskStr, "|", 2)
	if len(parts) != 2 {
		log.Printf("Invalid task format: %s", taskStr)
		w.redisClient.ZRem(ctx, "ping_queue", taskStr)
		return
	}

	taskIDStr, url := parts[0], parts[1]
	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil {
		log.Printf("Invalid task ID: %s", taskIDStr)
		w.redisClient.ZRem(ctx, "ping_queue", taskStr)
		return
	}

	start := time.Now()
	resp, err := http.Get(url)
	duration := time.Since(start).Milliseconds()

	if err := w.logRepo.TrimLogs(uint(taskID), 10); err != nil {
		log.Printf("Error trimming logs: %v", err)
	}

	if err != nil || resp.StatusCode != http.StatusOK {
		w.handleFailure(ctx, uint(taskID), url, duration, err, resp)
	} else {
		w.handleSuccess(ctx, uint(taskID), url, duration, resp.StatusCode)
		resp.Body.Close()
	}
}

func (w *PingWorker) handleSuccess(ctx context.Context, taskID uint, url string, duration int64, statusCode int) {
	newLog := &models.Log{
		TaskID:      taskID,
		Time:        time.Now(),
		TimeTake:    duration,
		LogResponse: "Successfully pinged",
		IsSuccess:   true,
		RespCode:    statusCode,
	}
	w.logRepo.Create(newLog)

	nextPing := time.Now().Add(10 * time.Minute).Unix()
	taskMember := fmt.Sprintf("%d|%s", taskID, url)
	w.redisClient.ZAdd(ctx, "ping_queue", &redis.Z{
		Score:  float64(nextPing),
		Member: taskMember,
	})
}

func (w *PingWorker) handleFailure(ctx context.Context, taskID uint, url string, duration int64, reqErr error, resp *http.Response) {
	statusCode := 0
	logMsg := "Failed to ping URL"
	if resp != nil {
		statusCode = resp.StatusCode
		resp.Body.Close()
	}
	if reqErr != nil {
		logMsg = reqErr.Error()
	}

	newLog := &models.Log{
		TaskID:      taskID,
		Time:        time.Now(),
		TimeTake:    duration,
		LogResponse: logMsg,
		IsSuccess:   false,
		RespCode:    statusCode,
	}
	w.logRepo.Create(newLog)

	var task models.Task
	if err := w.db.First(&task, taskID).Error; err != nil {
		log.Printf("Task not found %d: %v", taskID, err)
		return
	}

	task.FailCount++
	if task.FailCount >= 2 {
		task.IsActive = false
		w.db.Save(&task)
		
		taskMember := fmt.Sprintf("%d|%s", taskID, url)
		w.redisClient.ZRem(ctx, "ping_queue", taskMember)

		w.redisClient.LPush(ctx, "noti_queue", taskID)
	} else {
		w.db.Model(&task).Update("fail_count", task.FailCount)
		
		nextPing := time.Now().Add(10 * time.Minute).Unix()
		taskMember := fmt.Sprintf("%d|%s", taskID, url)
		w.redisClient.ZAdd(ctx, "ping_queue", &redis.Z{
			Score:  float64(nextPing),
			Member: taskMember,
		})
	}
}
