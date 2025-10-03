package services

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"task_manager/internal/models"
	"time"
)

type AIProcessor interface {
	ParseOrderMessage(message string) (*models.Order, []models.OrderItem, error)
	ParseTaskMessage(message string) (*models.Task, error)
	ExtractOrderItems(message string) ([]models.OrderItem, error)
}

type aiProcessor struct{}

func NewAIProcessor() AIProcessor {
	return &aiProcessor{}
}

// ParseOrderMessage processes natural language order messages
func (a *aiProcessor) ParseOrderMessage(message string) (*models.Order, []models.OrderItem, error) {
	// Extract order information using regex patterns
	order := &models.Order{}
	var items []models.OrderItem{}
	
	// Extract total amount
	totalRegex := regexp.MustCompile(`(?i)total[:\s]*(\d+(?:\.\d+)?)`)
	if matches := totalRegex.FindStringSubmatch(message); len(matches) > 1 {
		if total, err := strconv.ParseFloat(matches[1], 64); err == nil {
			order.TotalAmount = total
		}
	}
	
	// Extract customer name
	customerRegex := regexp.MustCompile(`(?i)customer[:\s]*([a-zA-Z\s]+)`)
	if matches := customerRegex.FindStringSubmatch(message); len(matches) > 1 {
		order.CustomerName = strings.TrimSpace(matches[1])
	}
	
	// Extract items
	items, err := a.ExtractOrderItems(message)
	if err != nil {
		return nil, nil, err
	}
	
	// Set default values
	order.Status = string(models.Pending)
	order.OrderDate = time.Now()
	
	return order, items, nil
}

// ExtractOrderItems extracts order items from natural language
func (a *aiProcessor) ExtractOrderItems(message string) ([]models.OrderItem, error) {
	var items []models.OrderItem
	
	// Pattern to match: "item name, qty X x price"
	itemRegex := regexp.MustCompile(`(?i)([a-zA-Z\s]+),\s*qty\s*(\d+)\s*x\s*(\d+(?:\.\d+)?)`)
	matches := itemRegex.FindAllStringSubmatch(message, -1)
	
	for i, match := range matches {
		if len(match) < 4 {
			continue
		}
		
		itemName := strings.TrimSpace(match[1])
		quantity, err := strconv.Atoi(match[2])
		if err != nil {
			continue
		}
		
		unitPrice, err := strconv.ParseFloat(match[3], 64)
		if err != nil {
			continue
		}
		
		totalPrice := float64(quantity) * unitPrice
		
		item := models.OrderItem{
			ItemName:   itemName,
			Quantity:   quantity,
			UnitPrice:  unitPrice,
			TotalPrice: totalPrice,
		}
		
		items = append(items, item)
	}
	
	return items, nil
}

// ParseTaskMessage processes natural language task messages
func (a *aiProcessor) ParseTaskMessage(message string) (*models.Task, error) {
	task := &models.Task{}
	
	// Extract task title (first few words)
	words := strings.Fields(message)
	if len(words) > 0 {
		task.Title = words[0]
		if len(words) > 1 {
			task.Description = strings.Join(words[1:], " ")
		}
	}
	
	// Set default values
	task.Status = string(models.Pending)
	task.Priority = string(models.Medium)
	task.CompletionPercentage = 0
	task.IsImplemented = false
	
	return task, nil
}

// ProcessWhatsAppMessage processes incoming WhatsApp messages with AI
func (a *aiProcessor) ProcessWhatsAppMessage(message string) (string, interface{}, error) {
	message = strings.ToLower(strings.TrimSpace(message))
	
	// Check if it's an order message
	if strings.Contains(message, "order") || strings.Contains(message, "total") {
		order, items, err := a.ParseOrderMessage(message)
		if err != nil {
			return "order", nil, err
		}
		
		result := map[string]interface{}{
			"order": order,
			"items": items,
		}
		
		return "order", result, nil
	}
	
	// Check if it's a task message
	if strings.Contains(message, "task") || strings.Contains(message, "create") {
		task, err := a.ParseTaskMessage(message)
		if err != nil {
			return "task", nil, err
		}
		
		return "task", task, nil
	}
	
	return "unknown", nil, fmt.Errorf("unable to process message type")
}
