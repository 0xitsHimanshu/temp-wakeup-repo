package repository

import (
	"upbot-server-go/internal/models"

	"gorm.io/gorm"
)

type LogRepository interface {
	Create(log *models.Log) error
	TrimLogs(taskID uint, maxLogs int) error
}

type logRepository struct {
	db *gorm.DB
}

func NewLogRepository(db *gorm.DB) LogRepository {
	return &logRepository{db: db}
}

func (r *logRepository) Create(log *models.Log) error {
	return r.db.Create(log).Error
}

func (r *logRepository) TrimLogs(taskID uint, maxLogs int) error {
	var logCount int64
	r.db.Model(&models.Log{}).Where("task_id = ?", taskID).Count(&logCount)
	if logCount >= int64(maxLogs) {
		// Delete the oldest logs, keeping only maxLogs-1 so we can add one more
		// Or just delete the oldest one. The original code deleted 1.
		// Let's be robust: delete any logs that are outside the latest (maxLogs - 1)
		// But for simplicity and matching original logic:
		return r.db.Where("task_id = ?", taskID).
			Order("time ASC").
			Limit(1).
			Delete(&models.Log{}).Error
	}
	return nil
}
