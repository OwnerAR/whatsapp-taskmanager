package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	ProcessWhatsAppMessage(message string) (string, interface{}, error)
	ProcessWithOpenAI(message string) (string, interface{}, error)
}

type aiProcessor struct {
	apiKey string
}

func NewAIProcessor(apiKey string) AIProcessor {
	return &aiProcessor{apiKey: apiKey}
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

// ProcessWithOpenAI processes messages using OpenAI API
func (a *aiProcessor) ProcessWithOpenAI(message string) (string, interface{}, error) {
	if a.apiKey == "" || a.apiKey == "your_openai_api_key" {
		// Fallback to regex processing if no API key
		return a.ProcessWhatsAppMessage(message)
	}

	// OpenAI API request
	requestBody := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{
				"role": "system",
				"content": `You are an AI assistant that processes WhatsApp messages for a task management system. 
				Extract structured data from natural language messages.
				
				For orders, extract: customer_name, total_amount, and items with quantity and price.
				For tasks, extract: title, description, assigned_to, priority.
				
				Return JSON format only.`,
			},
			{
				"role": "user",
				"content": message,
			},
		},
		"max_tokens": 500,
		"temperature": 0.1,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", nil, err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	var openAIResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &openAIResponse); err != nil {
		return "", nil, err
	}

	if len(openAIResponse.Choices) == 0 {
		return "", nil, fmt.Errorf("no response from OpenAI")
	}

	// Parse the AI response
	content := openAIResponse.Choices[0].Message.Content
	
	// Try to determine if it's an order or task based on content
	if strings.Contains(strings.ToLower(content), "order") || strings.Contains(strings.ToLower(content), "total") {
		return "order", content, nil
	} else if strings.Contains(strings.ToLower(content), "task") || strings.Contains(strings.ToLower(content), "create") {
		return "task", content, nil
	}

	return "unknown", content, nil
}
