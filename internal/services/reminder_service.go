package services

import (
	"task_manager/internal/models"
	"task_manager/internal/repository"
	"time"
	"fmt"
)

type ReminderService interface {
	CreateReminder(reminder *models.Reminder) error
	GetRemindersByTask(taskID uint) ([]models.Reminder, error)
	GetPendingReminders() ([]models.Reminder, error)
	UpdateReminder(reminder *models.Reminder) error
	DeleteReminder(id uint) error
	MarkReminderAsSent(id uint) error
	ProcessPendingReminders() error
	CreateTaskReminder(taskID uint, reminderType string, scheduledTime time.Time) error
	SendDailyProgressReminder(userPhone string, progress int) error
	SendMonthlyProgressReminder(userPhone string, progress int) error
func (s *reminderService) SendDailyProgressReminder(userPhone string, progress int) error {
	message := fmt.Sprintf("📅 Daily Progress Reminder: %d%% completed", progress)
	return s.whatsappService.SendMessage(userPhone, message)
}

func (s *reminderService) SendMonthlyProgressReminder(userPhone string, progress int) error {
	message := fmt.Sprintf("📆 Monthly Progress Reminder: %d%% completed", progress)
	return s.whatsappService.SendMessage(userPhone, message)
}
}

type reminderService struct {
	reminderRepo    repository.ReminderRepository
	whatsappService WhatsAppService
}

func NewReminderService(reminderRepo repository.ReminderRepository, whatsappService WhatsAppService) ReminderService {
	return &reminderService{
		reminderRepo:    reminderRepo,
		whatsappService: whatsappService,
	}
}

func (s *reminderService) CreateReminder(reminder *models.Reminder) error {
	return s.reminderRepo.Create(reminder)
}

func (s *reminderService) GetRemindersByTask(taskID uint) ([]models.Reminder, error) {
	return s.reminderRepo.GetByTaskID(taskID)
}

func (s *reminderService) GetPendingReminders() ([]models.Reminder, error) {
	return s.reminderRepo.GetPendingReminders()
}

func (s *reminderService) UpdateReminder(reminder *models.Reminder) error {
	return s.reminderRepo.Update(reminder)
}

func (s *reminderService) DeleteReminder(id uint) error {
	return s.reminderRepo.Delete(id)
}

func (s *reminderService) MarkReminderAsSent(id uint) error {
	return s.reminderRepo.MarkAsSent(id)
}

func (s *reminderService) ProcessPendingReminders() error {
	reminders, err := s.GetPendingReminders()
	if err != nil {
		return err
	}

	for _, reminder := range reminders {
		// Get task to find assigned user
		// taskID := reminder.TaskID
		// Assume we have a method to get task by ID (add to TaskService if needed)
		var phone string
		// var userName string
		if s.whatsappService != nil {
			// Try to get user WhatsApp number from task
			// This is a simplified logic, you may want to add GetTaskByID to TaskService
			// For now, just send to a placeholder or skip if not found
			// You can improve this by injecting TaskService to ReminderService
			phone = "" // TODO: get phone from assigned user
		}
		message := "🔔 Reminder: " + reminder.ReminderType
		if phone != "" {
			_ = s.whatsappService.SendMessage(phone, message)
		}
		// Mark as sent
		_ = s.MarkReminderAsSent(reminder.ID)
	}
	return nil
}
// Notifikasi progres harian/bulanan
func (s *reminderService) SendDailyProgressReminder(userPhone string, progress int) error {
	message := fmt.Sprintf("📅 Daily Progress Reminder: %d%% completed", progress)
	return s.whatsappService.SendMessage(userPhone, message)
}

func (s *reminderService) SendMonthlyProgressReminder(userPhone string, progress int) error {
	message := fmt.Sprintf("📆 Monthly Progress Reminder: %d%% completed", progress)
	return s.whatsappService.SendMessage(userPhone, message)
}


func (s *reminderService) CreateTaskReminder(taskID uint, reminderType string, scheduledTime time.Time) error {
	reminder := &models.Reminder{
		TaskID:        taskID,
		ReminderType:  reminderType,
		ScheduledTime: scheduledTime,
		WhatsAppSent:  false,
		CreatedAt:     time.Now(),
	}

	return s.CreateReminder(reminder)
}
