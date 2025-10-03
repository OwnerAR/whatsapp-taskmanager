package models

import (
	"time"

	"gorm.io/gorm"
)

type Task struct {
	ID                   uint           `json:"id" gorm:"primaryKey"`
	Title                string         `json:"title" gorm:"not null"`
	Description          string         `json:"description"`
	AssignedTo           uint           `json:"assigned_to" gorm:"not null"`
	DueDate              *time.Time    `json:"due_date"`
	Status               string         `json:"status" gorm:"default:'pending'"` // pending, in_progress, completed, overdue
	Priority             string         `json:"priority" gorm:"default:'medium'"` // low, medium, high, urgent
	CompletionPercentage int            `json:"completion_percentage" gorm:"default:0"`
	IsImplemented        bool           `json:"is_implemented" gorm:"default:false"`
	ImplementationNotes  string         `json:"implementation_notes"`
	TaskType             string         `json:"task_type" gorm:"default:'custom'"` // daily, monthly, custom
	IsRecurring          bool           `json:"is_recurring" gorm:"default:false"`
	RecurringPattern     string         `json:"recurring_pattern"` // daily, monthly
	LastUpdatedDate      *time.Time     `json:"last_updated_date"`
	CompletedAt          *time.Time     `json:"completed_at"`
	CreatedBy            uint           `json:"created_by" gorm:"not null"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

type TaskStatus string

const (
	Pending     TaskStatus = "pending"
	InProgress  TaskStatus = "in_progress"
	Completed   TaskStatus = "completed"
	Overdue     TaskStatus = "overdue"
)

type TaskPriority string

const (
	Low    TaskPriority = "low"
	Medium TaskPriority = "medium"
	High   TaskPriority = "high"
	Urgent TaskPriority = "urgent"
)

type TaskType string

const (
	Daily   TaskType = "daily"
	Monthly TaskType = "monthly"
	Custom  TaskType = "custom"
)

type TaskProgress struct {
	ID                   uint      `json:"id" gorm:"primaryKey"`
	TaskID               uint      `json:"task_id" gorm:"not null"`
	CompletionPercentage int       `json:"completion_percentage"`
	IsImplemented        bool      `json:"is_implemented"`
	ImplementationNotes  string    `json:"implementation_notes"`
	UpdatedBy            uint      `json:"updated_by"`
	UpdatedAt            time.Time `json:"updated_at"`
	CreatedAt            time.Time `json:"created_at"`
}

type DailyTask struct {
	ID                   uint      `json:"id" gorm:"primaryKey"`
	TaskID               uint      `json:"task_id" gorm:"not null"`
	TaskDate             time.Time `json:"task_date" gorm:"type:date"`
	CompletionPercentage int       `json:"completion_percentage"`
	IsImplemented        bool      `json:"is_implemented"`
	ImplementationNotes  string    `json:"implementation_notes"`
	UpdatedBy            uint      `json:"updated_by"`
	UpdatedAt            time.Time `json:"updated_at"`
	CreatedAt            time.Time `json:"created_at"`
}

type MonthlyTask struct {
	ID                   uint      `json:"id" gorm:"primaryKey"`
	TaskID               uint      `json:"task_id" gorm:"not null"`
	MonthYear            string    `json:"month_year" gorm:"type:varchar(7)"` // YYYY-MM
	CompletionPercentage int       `json:"completion_percentage"`
	IsImplemented        bool      `json:"is_implemented"`
	ImplementationNotes  string    `json:"implementation_notes"`
	UpdatedBy            uint      `json:"updated_by"`
	UpdatedAt            time.Time `json:"updated_at"`
	CreatedAt            time.Time `json:"created_at"`
}
