package models

import (
	"time"

	"gorm.io/gorm"
)

type Order struct {
	ID                    uint           `json:"id" gorm:"primaryKey"`
	OrderNumber           string         `json:"order_number" gorm:"unique;not null"`
	CustomerName          string         `json:"customer_name" gorm:"not null"`
	CustomerPhone         string         `json:"customer_phone"`
	OrderDate             time.Time      `json:"order_date" gorm:"not null"`
	DeliveryDate          *time.Time      `json:"delivery_date"`
	Status                string         `json:"status" gorm:"default:'pending'"` // pending, processing, completed, cancelled
	TotalAmount           float64        `json:"total_amount" gorm:"not null"`
	TaxPercentage         float64        `json:"tax_percentage"`
	TaxAmount             float64        `json:"tax_amount"`
	MarketingPercentage   float64        `json:"marketing_percentage"`
	MarketingCost         float64        `json:"marketing_cost"`
	RentalPercentage      float64        `json:"rental_percentage"`
	RentalCost            float64        `json:"rental_cost"`
	NetProfit             float64        `json:"net_profit"`
	CalculationTimestamp  time.Time      `json:"calculation_timestamp"`
	CreatedBy             uint           `json:"created_by" gorm:"not null"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

type OrderStatus string

const (
	OrderPending    OrderStatus = "pending"
	OrderProcessing OrderStatus = "processing"
	OrderCompleted  OrderStatus = "completed"
	OrderCancelled  OrderStatus = "cancelled"
)
