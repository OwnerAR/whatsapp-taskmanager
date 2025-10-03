package repository

import (
	"task_manager/internal/models"

	"gorm.io/gorm"
)

type OrderItemRepository interface {
	Create(orderItem *models.OrderItem) error
	GetByID(id uint) (*models.OrderItem, error)
	GetByOrderID(orderID uint) ([]*models.OrderItem, error)
	Update(orderItem *models.OrderItem) error
	Delete(id uint) error
	GetAll() ([]*models.OrderItem, error)
	GetByStatus(status string) ([]*models.OrderItem, error)
}

type orderItemRepository struct {
	db *gorm.DB
}

func NewOrderItemRepository(db *gorm.DB) OrderItemRepository {
	return &orderItemRepository{db: db}
}

func (r *orderItemRepository) Create(orderItem *models.OrderItem) error {
	return r.db.Create(orderItem).Error
}

func (r *orderItemRepository) GetByID(id uint) (*models.OrderItem, error) {
	var orderItem models.OrderItem
	err := r.db.First(&orderItem, id).Error
	if err != nil {
		return nil, err
	}
	return &orderItem, nil
}

func (r *orderItemRepository) GetByOrderID(orderID uint) ([]*models.OrderItem, error) {
	var orderItems []*models.OrderItem
	err := r.db.Where("order_id = ?", orderID).Find(&orderItems).Error
	if err != nil {
		return nil, err
	}
	return orderItems, nil
}

func (r *orderItemRepository) Update(orderItem *models.OrderItem) error {
	return r.db.Save(orderItem).Error
}

func (r *orderItemRepository) Delete(id uint) error {
	return r.db.Delete(&models.OrderItem{}, id).Error
}

func (r *orderItemRepository) GetAll() ([]*models.OrderItem, error) {
	var orderItems []*models.OrderItem
	err := r.db.Find(&orderItems).Error
	if err != nil {
		return nil, err
	}
	return orderItems, nil
}

func (r *orderItemRepository) GetByStatus(status string) ([]*models.OrderItem, error) {
	var orderItems []*models.OrderItem
	err := r.db.Where("status = ?", status).Find(&orderItems).Error
	if err != nil {
		return nil, err
	}
	return orderItems, nil
}
