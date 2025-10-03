package services

import (
	"fmt"
	"task_manager/internal/models"
	"task_manager/internal/repository"
	"task_manager/internal/redis"
	"time"
)

type TaskService interface {
	CreateTask(task *models.Task) error
	GetTaskByID(id uint) (*models.Task, error)
	GetTasksByUser(userID uint) ([]models.Task, error)
	GetDailyTasks(userID uint, date time.Time) ([]models.Task, error)
	GetMonthlyTasks(userID uint, monthYear string) ([]models.Task, error)
	UpdateTask(task *models.Task) error
	UpdateTaskProgress(taskID uint, progress int, isImplemented bool, notes string, updatedBy uint) error
	DeleteTask(id uint) error
	CreateDailyTask(task *models.Task) error
	CreateMonthlyTask(task *models.Task) error
	ResetDailyTasks() error
	ResetMonthlyTasks() error
}

type taskService struct {
	taskRepo repository.TaskRepository
	redis    *redis.Client
}

func NewTaskService(taskRepo repository.TaskRepository, redis *redis.Client) TaskService {
	return &taskService{taskRepo: taskRepo, redis: redis}
}

func (s *taskService) CreateTask(task *models.Task) error {
	return s.taskRepo.Create(task)
}

func (s *taskService) GetTaskByID(id uint) (*models.Task, error) {
	return s.taskRepo.GetByID(id)
}

func (s *taskService) GetTasksByUser(userID uint) ([]models.Task, error) {
	return s.taskRepo.GetByUserID(userID)
}

func (s *taskService) GetDailyTasks(userID uint, date time.Time) ([]models.Task, error) {
	return s.taskRepo.GetDailyTasks(userID, date)
}

func (s *taskService) GetMonthlyTasks(userID uint, monthYear string) ([]models.Task, error) {
	return s.taskRepo.GetMonthlyTasks(userID, monthYear)
}

func (s *taskService) UpdateTask(task *models.Task) error {
	return s.taskRepo.Update(task)
}

func (s *taskService) UpdateTaskProgress(taskID uint, progress int, isImplemented bool, notes string, updatedBy uint) error {
	// Update in database
	err := s.taskRepo.UpdateProgress(taskID, progress, isImplemented, notes, updatedBy)
	if err != nil {
		return err
	}

	// Cache progress in Redis
	ttl := time.Hour * 24 // 24 hours
	return s.redis.SetTaskProgress(taskID, progress, ttl)
}

func (s *taskService) DeleteTask(id uint) error {
	return s.taskRepo.Delete(id)
}

func (s *taskService) CreateDailyTask(task *models.Task) error {
	task.TaskType = string(models.Daily)
	task.IsRecurring = true
	task.RecurringPattern = "daily"
	return s.taskRepo.Create(task)
}

func (s *taskService) CreateMonthlyTask(task *models.Task) error {
	task.TaskType = string(models.Monthly)
	task.IsRecurring = true
	task.RecurringPattern = "monthly"
	return s.taskRepo.Create(task)
}

func (s *taskService) ResetDailyTasks() error {
	// This would be called by a cron job or scheduler
	// Reset all daily tasks to 0% completion
	// Implementation depends on your specific requirements
	return nil
}

func (s *taskService) ResetMonthlyTasks() error {
	// This would be called by a cron job or scheduler
	// Reset all monthly tasks to 0% completion
	// Implementation depends on your specific requirements
	return nil
}
