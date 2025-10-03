package services

import (
	"fmt"
	"task_manager/internal/redis"
	"task_manager/pkg/whatsapp"
	"time"
)

type WhatsAppService interface {
	SendMessage(phone, message string) error
	SendForwardedMessage(phone, message string, duration int) error
	StartInteractiveSession(userID uint, phoneNumber, command string) (string, error)
	UpdateSession(sessionID string, data *redis.SessionData) error
	GetSession(sessionID string) (*redis.SessionData, error)
	EndSession(sessionID string) error
	SetTempData(key string, value interface{}, ttl time.Duration) error
	GetTempData(key string, dest interface{}) error
	DeleteTempData(key string) error
}

type whatsappService struct {
	client *whatsapp.Client
	redis  *redis.Client
}

func NewWhatsAppService(client *whatsapp.Client, redis *redis.Client) WhatsAppService {
	return &whatsappService{client: client, redis: redis}
}

func (s *whatsappService) SendMessage(phone, message string) error {
	return s.client.SendTextMessage(phone, message)
}

func (s *whatsappService) SendForwardedMessage(phone, message string, duration int) error {
	return s.client.SendForwardedMessage(phone, message, duration)
}

func (s *whatsappService) StartInteractiveSession(userID uint, phoneNumber, command string) (string, error) {
	// Generate session ID
	sessionID := fmt.Sprintf("session_%d_%d", userID, time.Now().Unix())
	
	// Create session data
	sessionData := &redis.SessionData{
		UserID:      userID,
		PhoneNumber: phoneNumber,
		Command:     command,
		Step:        1,
		Data:        make(map[string]interface{}),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	// Store session in Redis
	ttl := time.Duration(3600) * time.Second // 1 hour
	err := s.redis.SetSession(sessionID, sessionData, ttl)
	if err != nil {
		return "", err
	}
	
	return sessionID, nil
}

func (s *whatsappService) UpdateSession(sessionID string, data *redis.SessionData) error {
	ttl := time.Duration(3600) * time.Second // 1 hour
	return s.redis.UpdateSession(sessionID, data, ttl)
}

func (s *whatsappService) GetSession(sessionID string) (*redis.SessionData, error) {
	return s.redis.GetSession(sessionID)
}

func (s *whatsappService) EndSession(sessionID string) error {
	return s.redis.DeleteSession(sessionID)
}

func (s *whatsappService) SetTempData(key string, value interface{}, ttl time.Duration) error {
	return s.redis.SetTempData(key, value, ttl)
}

func (s *whatsappService) GetTempData(key string, dest interface{}) error {
	return s.redis.GetTempData(key, dest)
}

func (s *whatsappService) DeleteTempData(key string) error {
	return s.redis.DeleteTempData(key)
}
