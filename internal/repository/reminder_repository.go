package repository

import (
	"task_manager/internal/models"
	"time"

	"gorm.io/gorm"
)

type ReminderRepository interface {
	Create(reminder *models.Reminder) error
	GetByTaskID(taskID uint) ([]models.Reminder, error)
	GetPendingReminders() ([]models.Reminder, error)
	Update(reminder *models.Reminder) error
	Delete(id uint) error
	MarkAsSent(id uint) error
}

type reminderRepository struct {
	db *gorm.DB
}

func NewReminderRepository(db *gorm.DB) ReminderRepository {
	return &reminderRepository{db: db}
}

func (r *reminderRepository) Create(reminder *models.Reminder) error {
	return r.db.Create(reminder).Error
}

func (r *reminderRepository) GetByTaskID(taskID uint) ([]models.Reminder, error) {
	var reminders []models.Reminder
	err := r.db.Where("task_id = ?", taskID).Find(&reminders).Error
	return reminders, err
}

func (r *reminderRepository) GetPendingReminders() ([]models.Reminder, error) {
	var reminders []models.Reminder
	err := r.db.Where("whatsapp_sent = ? AND scheduled_time <= ?", false, time.Now()).Find(&reminders).Error
	return reminders, err
}

func (r *reminderRepository) Update(reminder *models.Reminder) error {
	return r.db.Save(reminder).Error
}

func (r *reminderRepository) Delete(id uint) error {
	return r.db.Delete(&models.Reminder{}, id).Error
}

func (r *reminderRepository) MarkAsSent(id uint) error {
	return r.db.Model(&models.Reminder{}).Where("id = ?", id).Update("whatsapp_sent", true).Error
}
