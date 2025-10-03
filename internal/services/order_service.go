package services

import (
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
}

type orderService struct {
	orderRepo    repository.OrderRepository
	financialRepo repository.FinancialRepository
}

func NewOrderService(orderRepo repository.OrderRepository, financialRepo repository.FinancialRepository) OrderService {
	return &orderService{orderRepo: orderRepo, financialRepo: financialRepo}
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
