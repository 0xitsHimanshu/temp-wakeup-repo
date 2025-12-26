package worker

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"upbot-server-go/database"
	"upbot-server-go/libraries"
	"upbot-server-go/models"

	"github.com/go-redis/redis/v8"
)

// PerformPing executes an immediate ping for a given task
func PerformPing(taskID uint, url string) {
	log.Printf("Performing immediate ping for task %d: %s", taskID, url)
	
	redisClient := libraries.GetInstance()
	taskMember := fmt.Sprintf("%d|%s", taskID, url)
	
	if err := TrimLogs(taskID); err != nil {
		log.Printf("Error trimming logs for task %d: %v", taskID, err)
	}
	
	timeNow := time.Now()
	resp, err := http.Get(url)
	timeSince := time.Since(timeNow).Milliseconds()
	
	if err != nil {
		newLog := models.Log{
			LogResponse: "Failed to ping URL",
			Time:        time.Now(),
			TimeTake:    int64(timeSince),
			TaskID:      taskID,
			IsSuccess:   false,
		}
		if err := database.DB.Create(&newLog).Error; err != nil {
			log.Printf("Error creating log: %v", err)
		}
		log.Printf("Failed to ping %s: %v", url, err)
		return
	}
	defer resp.Body.Close()
	
	// Consider 2xx and 3xx status codes as success (server is responding)
	isSuccess := resp.StatusCode >= 200 && resp.StatusCode < 400
	
	if !isSuccess {
		newLog := models.Log{
			LogResponse: fmt.Sprintf("Server responded with status: %d", resp.StatusCode),
			Time:        time.Now(),
			TimeTake:    int64(timeSince),
			TaskID:      taskID,
			IsSuccess:   false,
			RespCode:    resp.StatusCode,
		}
		if err := database.DB.Create(&newLog).Error; err != nil {
			log.Printf("Error creating log: %v", err)
		}
		
		var task models.Task
		if err := database.DB.First(&task, taskID).Error; err == nil {
			task.FailCount++
			if task.FailCount >= 2 {
				task.IsActive = false
				database.DB.Save(&task)
				redisClient.ZRem(context.Background(), "ping_queue", taskMember)
				redisClient.LPush(context.Background(), "noti_queue", taskID)
			} else {
				database.DB.Model(&task).Update("fail_count", task.FailCount)
				nextPing := time.Now().Add(10 * time.Minute).Unix()
				_, err = redisClient.ZAdd(context.Background(), "ping_queue", &redis.Z{
					Score:  float64(nextPing),
					Member: taskMember,
				}).Result()
				if err != nil {
					log.Printf("Error rescheduling URL %s: %v", url, err)
				}
			}
		} else {
			log.Printf("Error fetching task: %v", err)
		}
		log.Printf("Ping failed for %s with status code: %d", url, resp.StatusCode)
		return
	}
	
	// Success: Status code is 2xx or 3xx
	if isSuccess {
		nextPing := time.Now().Add(10 * time.Minute).Unix()
		_, err = redisClient.ZAdd(context.Background(), "ping_queue", &redis.Z{
			Score:  float64(nextPing),
			Member: taskMember,
		}).Result()
		// Reset fail count on success
		database.DB.Model(&models.Task{}).Where("id = ?", taskID).Update("fail_count", 0)
		
		newLog := models.Log{
			LogResponse: fmt.Sprintf("Server is up (Status: %d)", resp.StatusCode),
			Time:        time.Now(),
			TimeTake:    int64(timeSince),
			TaskID:      taskID,
			IsSuccess:   true,
			RespCode:    resp.StatusCode,
		}
		if err := database.DB.Create(&newLog).Error; err != nil {
			log.Printf("Error creating log: %v", err)
		}
		log.Printf("Successfully pinged %s - Status %d (took %dms)", url, resp.StatusCode, timeSince)
		if err := database.DB.Create(&newLog).Error; err != nil {
			log.Printf("Error creating log: %v", err)
		}
		log.Printf("Successfully pinged %s (took %dms)", url, timeSince)
	}
}

func StartPingWorker() {
	for {
		now := time.Now().Unix()
		redisClient := libraries.GetInstance()
		tasks, err := redisClient.ZRangeByScore(context.Background(), "ping_queue", &redis.ZRangeBy{
			Min: "-inf",
			Max: fmt.Sprintf("%d", now),
		}).Result()
		if err != nil {
			log.Printf("Error fetching from queue: %v", err)
			time.Sleep(1 * time.Minute)
			continue
		}
		for _, task := range tasks {
			parts := strings.SplitN(task, "|", 2)
			if len(parts) != 2 {
				log.Printf("Invalid task format: %s", task)
				continue
			}
			taskIdStr, url := parts[0], parts[1]
			taskId, err := strconv.Atoi(taskIdStr)
			if err != nil {
				log.Printf("Invalid task ID: %s", taskIdStr)
				continue
			}
			
			// Remove from queue before pinging to avoid duplicates
			redisClient.ZRem(context.Background(), "ping_queue", task)
			
			// Use the extracted ping function
			PerformPing(uint(taskId), url)
		}

		time.Sleep(10 * time.Minute)
	}
}

const MaxLogsPerTask = 10

func TrimLogs(taskID uint) error {
	var logCount int64
	database.DB.Model(&models.Log{}).Where("task_id = ?", taskID).Count(&logCount)
	if logCount >= MaxLogsPerTask {
		database.DB.Where("task_id = ?", taskID).
			Order("time ASC").
			Limit(1).
			Delete(&models.Log{})
	}
	return nil
}
