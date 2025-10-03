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
		// Send WhatsApp message
		message := "Reminder: " + reminder.ReminderType
		err := s.whatsappService.SendMessage("", message) // Phone number should be retrieved from task
		if err != nil {
			continue // Log error but continue with other reminders
		}

		// Mark as sent
		err = s.MarkReminderAsSent(reminder.ID)
		if err != nil {
			continue // Log error but continue
		}
	}

	return nil
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

func (s *reminderService) SendDailyProgressReminder(userPhone string, progress int) error {
	message := fmt.Sprintf("📅 Daily Progress Reminder: %d%% completed", progress)
	return s.whatsappService.SendMessage(userPhone, message)
}

func (s *reminderService) SendMonthlyProgressReminder(userPhone string, progress int) error {
	message := fmt.Sprintf("📆 Monthly Progress Reminder: %d%% completed", progress)
	return s.whatsappService.SendMessage(userPhone, message)
}