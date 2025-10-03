package services

import (
	"errors"
	"task_manager/internal/models"
	"task_manager/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	CreateUser(user *models.User, password string) error
	GetUserByID(id uint) (*models.User, error)
	GetUserByUsername(username string) (*models.User, error)
	GetUserByWhatsAppNumber(whatsappNumber string) (*models.User, error)
	GetAllUsers() ([]models.User, error)
	UpdateUser(user *models.User) error
	DeleteUser(id uint) error
	ValidateUserRole(userID uint, requiredRole string) error
}

type userService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{userRepo: userRepo}
}

func (s *userService) CreateUser(user *models.User, password string) error {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	
	// For now, we'll store password in a separate field if needed
	// In a real implementation, you might want to add a password field to User model
	_ = hashedPassword
	
	return s.userRepo.Create(user)
}

func (s *userService) GetUserByID(id uint) (*models.User, error) {
	return s.userRepo.GetByID(id)
}

func (s *userService) GetUserByUsername(username string) (*models.User, error) {
	return s.userRepo.GetByUsername(username)
}

func (s *userService) GetUserByWhatsAppNumber(whatsappNumber string) (*models.User, error) {
	return s.userRepo.GetByWhatsAppNumber(whatsappNumber)
}

func (s *userService) GetAllUsers() ([]models.User, error) {
	return s.userRepo.GetAll()
}

func (s *userService) UpdateUser(user *models.User) error {
	return s.userRepo.Update(user)
}

func (s *userService) DeleteUser(id uint) error {
	return s.userRepo.Delete(id)
}

func (s *userService) ValidateUserRole(userID uint, requiredRole string) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}
	
	// Check if user has required role
	if user.Role != requiredRole {
		return errors.New("insufficient permissions")
	}
	
	return nil
}
