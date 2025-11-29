package repository

import (
	"upbot-server-go/internal/models"

	"gorm.io/gorm"
)

// TaskRepository defines the interface for task-related database operations.
// This allows us to mock the repository in tests.
type TaskRepository interface {
	Create(task *models.Task) error
	CountActiveTasksByUserID(userID uint) (int64, error)
	FindByURLAndUserID(url string, userID uint) (*models.Task, error)
	GetUserByEmail(email string) (*models.User, error)
}

type taskRepository struct {
	db *gorm.DB
}

// NewTaskRepository creates a new instance of TaskRepository.
func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{db: db}
}

func (r *taskRepository) Create(task *models.Task) error {
	return r.db.Create(task).Error
}

func (r *taskRepository) CountActiveTasksByUserID(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.Task{}).Where("user_id = ? AND is_active = ?", userID, true).Count(&count).Error
	return count, err
}

func (r *taskRepository) FindByURLAndUserID(url string, userID uint) (*models.Task, error) {
	var task models.Task
	err := r.db.Where("url = ? AND user_id = ?", url, userID).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *taskRepository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Tasks").Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
