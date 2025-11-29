package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email string `json:"email" gorm:"uniqueIndex;not null"`
	Tasks []Task `json:"tasks" gorm:"foreignKey:UserID"`
}

type Task struct {
	gorm.Model
	URL           string  `json:"url" gorm:"not null"`
	IsActive      bool    `json:"isActive" gorm:"default:true"`
	NotifyDiscord bool    `json:"notifyDiscord" gorm:"default:false"`
	WebHook       *string `json:"webHook" gorm:"default:NULL"`
	UserID        uint    `json:"userId" gorm:"not null"`
	FailCount     int     `json:"failCount" gorm:"default:0"`
	// Logs are omitted from the main struct to avoid fetching them every time
}

type Log struct {
	gorm.Model
	TaskID      uint      `json:"taskId" gorm:"index"`
	Time        time.Time `json:"time"`
	TimeTake    int64     `json:"timeTake"`
	LogResponse string    `json:"logResponse"`
	IsSuccess   bool      `json:"isSuccess"`
	RespCode    int       `json:"respCode"`
}
