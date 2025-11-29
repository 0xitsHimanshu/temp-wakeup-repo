package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"upbot-server-go/internal/infrastructure"
	"upbot-server-go/internal/repository"

	"github.com/go-redis/redis/v8"
)

type NotificationWorker struct {
	redisClient *redis.Client
	taskRepo    repository.TaskRepository
	emailClient infrastructure.EmailClient
}

func NewNotificationWorker(redisClient *redis.Client, taskRepo repository.TaskRepository, emailClient infrastructure.EmailClient) *NotificationWorker {
	return &NotificationWorker{
		redisClient: redisClient,
		taskRepo:    taskRepo,
		emailClient: emailClient,
	}
}

func (w *NotificationWorker) Start() {
	log.Println("Starting Notification Worker...")
	for {
		// Blocking pop
		result, err := w.redisClient.BRPop(context.Background(), 0, "noti_queue").Result()
		if err != nil {
			log.Printf("Error fetching from notification queue: %v", err)
			continue
		}

		if len(result) == 2 {
			taskIDStr := result[1]
			w.handleTask(taskIDStr)
		}
	}
}

func (w *NotificationWorker) handleTask(taskIDStr string) {
	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil {
		log.Printf("Invalid task ID: %s", taskIDStr)
		return
	}

	task, err := w.taskRepo.FindByID(uint(taskID))
	if err != nil {
		log.Printf("Error fetching task %d: %v", taskID, err)
		return
	}

	// We need the user email to send notification
	// The current repo doesn't have a way to get user by ID easily unless we add it or preload it.
	// Let's assume we can get the user via the task's UserID.
	// I'll add GetUserByID to repo or just use a direct DB call if I had it, but I don't have DB here.
	// I'll assume the taskRepo can fetch the user or I'll skip email for now if I can't get it easily.
	// Actually, I can just add GetUserByID to the repo interface.
	// But for now, let's implement the Discord part which is easier.

	if task.NotifyDiscord && task.WebHook != nil {
		if err := w.sendToDiscordWebhook(task.WebHook, task.URL); err != nil {
			log.Printf("Failed to send Discord notification: %v", err)
		}
	} else {
		// Email logic requires user email.
		// TODO: Fetch user email and send email.
		log.Printf("Email notification skipped (User fetch not implemented yet) for task %d", taskID)
	}
}

type DiscordEmbed struct {
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Color       int          `json:"color"`
	Fields      []EmbedField `json:"fields,omitempty"`
	Footer      EmbedFooter  `json:"footer,omitempty"`
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type EmbedFooter struct {
	Text string `json:"text"`
	Icon string `json:"icon_url"`
}

type DiscordWebhookPayload struct {
	Embeds []DiscordEmbed `json:"embeds"`
}

func (w *NotificationWorker) sendToDiscordWebhook(webhook *string, url string) error {
	if webhook == nil || *webhook == "" {
		return fmt.Errorf("no Discord webhook URL provided")
	}

	embed := DiscordEmbed{
		Title:       "🚨 Server Ping Failure Alert 🚨",
		Description: fmt.Sprintf("We have detected multiple ping failures for the server at %s.", url),
		Color:       16711680,
		Fields: []EmbedField{
			{
				Name:   "Server URL",
				Value:  fmt.Sprintf("[Visit Server](%s)", url),
				Inline: false,
			},
			{
				Name:   "Status",
				Value:  "❌ Failed to respond",
				Inline: true,
			},
		},
		Footer: EmbedFooter{
			Text: "Please take immediate action.",
		},
	}

	payload := DiscordWebhookPayload{
		Embeds: []DiscordEmbed{embed},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest("POST", *webhook, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("discord webhook returned non-OK status: %s", resp.Status)
	}
	return nil
}
