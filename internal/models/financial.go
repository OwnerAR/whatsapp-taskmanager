package models

import (
	"time"

	"gorm.io/gorm"
)

type Reminder struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	TaskID       uint           `json:"task_id" gorm:"not null"`
	ReminderType string         `json:"reminder_type" gorm:"not null"`
	ScheduledTime time.Time     `json:"scheduled_time" gorm:"not null"`
	WhatsAppSent bool           `json:"whatsapp_sent" gorm:"default:false"`
	CreatedAt    time.Time      `json:"created_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

type FinancialSettings struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	SettingName    string    `json:"setting_name" gorm:"not null"` // tax_rate, marketing_rate, rental_rate
	PercentageValue float64   `json:"percentage_value"`
	FixedAmount    float64    `json:"fixed_amount"`
	IsPercentage   bool       `json:"is_percentage" gorm:"default:true"`
	IsActive       bool       `json:"is_active" gorm:"default:true"`
	CreatedBy      uint       `json:"created_by" gorm:"not null"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

type CalculationHistory struct {
	ID                    uint      `json:"id" gorm:"primaryKey"`
	OrderID               uint      `json:"order_id" gorm:"not null"`
	CalculationType       string    `json:"calculation_type" gorm:"not null"` // tax, marketing, rental, net_profit
	InputValue            float64   `json:"input_value"`
	PercentageUsed        float64   `json:"percentage_used"`
	CalculatedAmount      float64   `json:"calculated_amount"`
	CalculationTimestamp  time.Time `json:"calculation_timestamp"`
	CreatedAt             time.Time `json:"created_at"`
}

type ReportQuery struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       uint      `json:"user_id" gorm:"not null"`
	QueryType    string    `json:"query_type" gorm:"not null"` // daily, monthly, yearly, custom_range
	StartDate    *time.Time `json:"start_date"`
	EndDate      *time.Time `json:"end_date"`
	ReportData   string    `json:"report_data" gorm:"type:json"`
	GeneratedAt  time.Time `json:"generated_at"`
	CreatedAt    time.Time `json:"created_at"`
}
