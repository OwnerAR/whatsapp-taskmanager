package models

import (
	"time"

	"gorm.io/gorm"
)

type OrderItem struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	OrderID     uint           `json:"order_id" gorm:"not null"`
	ItemName    string         `json:"item_name" gorm:"not null"`
	Quantity    int            `json:"quantity" gorm:"not null"`
	UnitPrice   float64        `json:"unit_price" gorm:"not null"`
	TotalPrice  float64        `json:"total_price" gorm:"not null"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

// OrderItemStatus represents the status of an order item
type OrderItemStatus string

const (
	Pending   OrderItemStatus = "pending"
	Completed OrderItemStatus = "completed"
	Cancelled OrderItemStatus = "cancelled"
)
