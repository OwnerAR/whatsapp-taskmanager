package repository

import (
	"task_manager/internal/models"
	"time"

	"gorm.io/gorm"
)

type TaskRepository interface {
	Create(task *models.Task) error
	GetByID(id uint) (*models.Task, error)
	GetByUserID(userID uint) ([]models.Task, error)
	GetDailyTasks(userID uint, date time.Time) ([]models.Task, error)
	GetMonthlyTasks(userID uint, monthYear string) ([]models.Task, error)
	Update(task *models.Task) error
	Delete(id uint) error
	UpdateProgress(taskID uint, progress int, isImplemented bool, notes string, updatedBy uint) error
}

type taskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{db: db}
}

func (r *taskRepository) Create(task *models.Task) error {
	return r.db.Create(task).Error
}

func (r *taskRepository) GetByID(id uint) (*models.Task, error) {
	var task models.Task
	err := r.db.First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *taskRepository) GetByUserID(userID uint) ([]models.Task, error) {
	var tasks []models.Task
	err := r.db.Where("assigned_to = ?", userID).Find(&tasks).Error
	return tasks, err
}

func (r *taskRepository) GetDailyTasks(userID uint, date time.Time) ([]models.Task, error) {
	var tasks []models.Task
	err := r.db.Where("assigned_to = ? AND task_type = ?", userID, "daily").Find(&tasks).Error
	return tasks, err
}

func (r *taskRepository) GetMonthlyTasks(userID uint, monthYear string) ([]models.Task, error) {
	var tasks []models.Task
	err := r.db.Where("assigned_to = ? AND task_type = ?", userID, "monthly").Find(&tasks).Error
	return tasks, err
}

func (r *taskRepository) Update(task *models.Task) error {
	return r.db.Save(task).Error
}

func (r *taskRepository) Delete(id uint) error {
	return r.db.Delete(&models.Task{}, id).Error
}

func (r *taskRepository) UpdateProgress(taskID uint, progress int, isImplemented bool, notes string, updatedBy uint) error {
	now := time.Now()
	
	// Update main task
	err := r.db.Model(&models.Task{}).Where("id = ?", taskID).Updates(map[string]interface{}{
		"completion_percentage": progress,
		"is_implemented":        isImplemented,
		"implementation_notes": notes,
		"last_updated_date":     now,
		"updated_at":           now,
	}).Error
	
	if err != nil {
		return err
	}

	// Create progress record
	progressRecord := &models.TaskProgress{
		TaskID:               taskID,
		CompletionPercentage: progress,
		IsImplemented:        isImplemented,
		ImplementationNotes:  notes,
		UpdatedBy:            updatedBy,
		UpdatedAt:            now,
	}

	return r.db.Create(progressRecord).Error
}
