package services

import (
	"errors"
	"fmt"
	"task_manager/internal/models"
	"task_manager/internal/repository"
	"time"
)

type OrderService interface {
	CreateOrder(order *models.Order) error
	GetOrderByID(id uint) (*models.Order, error)
	GetOrdersByUser(userID uint) ([]models.Order, error)
	GetOrdersByDateRange(startDate, endDate time.Time) ([]models.Order, error)
	UpdateOrder(order *models.Order) error
	DeleteOrder(id uint) error
	CalculateFinancials(order *models.Order) error
	GetAllOrders() ([]models.Order, error)
	
	// Order Items methods
	AddItemToOrder(orderID uint, itemName string, quantity int, price float64, description string) error
	GetOrderItems(orderID uint) ([]*models.OrderItem, error)
	UpdateOrderItem(orderItem *models.OrderItem) error
	DeleteOrderItem(itemID uint) error
	UpdateItemStatus(itemID uint, status string) error
	GetOrderItemsSummary(orderID uint) (map[string]interface{}, error)
}

type orderService struct {
	orderRepo     repository.OrderRepository
	orderItemRepo repository.OrderItemRepository
	financialRepo repository.FinancialRepository
}

func NewOrderService(orderRepo repository.OrderRepository, orderItemRepo repository.OrderItemRepository, financialRepo repository.FinancialRepository) OrderService {
	return &orderService{orderRepo: orderRepo, orderItemRepo: orderItemRepo, financialRepo: financialRepo}
}

func (s *orderService) CreateOrder(order *models.Order) error {
	// Calculate financials before creating
	if err := s.CalculateFinancials(order); err != nil {
		return err
	}
	
	return s.orderRepo.Create(order)
}

func (s *orderService) GetOrderByID(id uint) (*models.Order, error) {
	return s.orderRepo.GetByID(id)
}

func (s *orderService) GetOrdersByUser(userID uint) ([]models.Order, error) {
	return s.orderRepo.GetByUserID(userID)
}

func (s *orderService) GetOrdersByDateRange(startDate, endDate time.Time) ([]models.Order, error) {
	return s.orderRepo.GetByDateRange(startDate, endDate)
}

func (s *orderService) UpdateOrder(order *models.Order) error {
	// Recalculate financials before updating
	if err := s.CalculateFinancials(order); err != nil {
		return err
	}
	
	return s.orderRepo.Update(order)
}

func (s *orderService) DeleteOrder(id uint) error {
	return s.orderRepo.Delete(id)
}

func (s *orderService) CalculateFinancials(order *models.Order) error {
	// Get financial settings
	taxSettings, err := s.financialRepo.GetSettings("tax_rate")
	if err != nil {
		return fmt.Errorf("failed to get tax settings: %w", err)
	}
	
	marketingSettings, err := s.financialRepo.GetSettings("marketing_rate")
	if err != nil {
		return fmt.Errorf("failed to get marketing settings: %w", err)
	}
	
	rentalSettings, err := s.financialRepo.GetSettings("rental_rate")
	if err != nil {
		return fmt.Errorf("failed to get rental settings: %w", err)
	}
	
	// Calculate tax amount
	order.TaxPercentage = taxSettings.PercentageValue
	order.TaxAmount = order.TotalAmount * (taxSettings.PercentageValue / 100)
	
	// Calculate marketing cost
	order.MarketingPercentage = marketingSettings.PercentageValue
	order.MarketingCost = order.TotalAmount * (marketingSettings.PercentageValue / 100)
	
	// Calculate rental cost
	order.RentalPercentage = rentalSettings.PercentageValue
	order.RentalCost = order.TotalAmount * (rentalSettings.PercentageValue / 100)
	
	// Calculate net profit
	order.NetProfit = order.TotalAmount - order.TaxAmount - order.MarketingCost - order.RentalCost
	
	// Set calculation timestamp
	order.CalculationTimestamp = time.Now()
	
	// Create calculation history
	history := &models.CalculationHistory{
		OrderID:              order.ID,
		CalculationType:      "net_profit",
		InputValue:           order.TotalAmount,
		PercentageUsed:       taxSettings.PercentageValue + marketingSettings.PercentageValue + rentalSettings.PercentageValue,
		CalculatedAmount:     order.NetProfit,
		CalculationTimestamp: time.Now(),
	}
	
	return s.financialRepo.CreateCalculationHistory(history)
}

func (s *orderService) GetAllOrders() ([]models.Order, error) {
	return s.orderRepo.GetAll()
}

// Order Items methods implementation

func (s *orderService) AddItemToOrder(orderID uint, itemName string, quantity int, price float64, description string) error {
	// Verify order exists
	order, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return err
	}
	if order == nil {
		return errors.New("order not found")
	}

	// Create order item
	orderItem := &models.OrderItem{
		OrderID:     orderID,
		ItemName:    itemName,
		Quantity:    quantity,
		UnitPrice:   price,
		TotalPrice:  float64(quantity) * price,
		Description: description,
		Status:      string(models.ItemPending),
	}

	return s.orderItemRepo.Create(orderItem)
}

func (s *orderService) GetOrderItems(orderID uint) ([]*models.OrderItem, error) {
	return s.orderItemRepo.GetByOrderID(orderID)
}

func (s *orderService) UpdateOrderItem(orderItem *models.OrderItem) error {
	return s.orderItemRepo.Update(orderItem)
}

func (s *orderService) DeleteOrderItem(itemID uint) error {
	return s.orderItemRepo.Delete(itemID)
}

func (s *orderService) UpdateItemStatus(itemID uint, status string) error {
	orderItem, err := s.orderItemRepo.GetByID(itemID)
	if err != nil {
		return err
	}
	if orderItem == nil {
		return errors.New("order item not found")
	}

	orderItem.Status = status
	return s.orderItemRepo.Update(orderItem)
}

func (s *orderService) GetOrderItemsSummary(orderID uint) (map[string]interface{}, error) {
	orderItems, err := s.orderItemRepo.GetByOrderID(orderID)
	if err != nil {
		return nil, err
	}

	totalItems := len(orderItems)
	totalQuantity := 0
	totalValue := 0.0
	pendingItems := 0
	completedItems := 0

	for _, item := range orderItems {
		totalQuantity += item.Quantity
		totalValue += item.TotalPrice
		
		if item.Status == string(models.ItemPending) {
			pendingItems++
		} else if item.Status == string(models.ItemCompleted) {
			completedItems++
		}
	}

	return map[string]interface{}{
		"total_items":      totalItems,
		"total_quantity":   totalQuantity,
		"total_value":      totalValue,
		"pending_items":    pendingItems,
		"completed_items":  completedItems,
		"completion_rate":  float64(completedItems) / float64(totalItems) * 100,
	}, nil
}
