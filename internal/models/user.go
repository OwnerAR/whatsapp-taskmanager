package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	Username      string         `json:"username" gorm:"unique;not null"`
	Email         string         `json:"email" gorm:"unique;not null"`
	PhoneNumber   string         `json:"phone_number"`
	Role          string         `json:"role" gorm:"default:'user'"` // super_admin, admin, user
	WhatsAppNumber string        `json:"whatsapp_number" gorm:"column:whats_app_number"`
	IsActive      bool           `json:"is_active" gorm:"default:true"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

type UserRole string

const (
	SuperAdmin UserRole = "super_admin"
	Admin      UserRole = "admin"
    Users UserRole = "user"
)
