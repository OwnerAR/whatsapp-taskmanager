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
	"task_manager/internal/redis"
	"time"
)

type AIProcessor interface {
	ParseOrderMessage(message string) (*models.Order, []models.OrderItem, error)
	ParseTaskMessage(message string) (*models.Task, error)
	ExtractOrderItems(message string) ([]models.OrderItem, error)
	ProcessWhatsAppMessage(message string) (string, interface{}, error)
	ProcessWithOpenAI(message string, userID string) (string, interface{}, error)
	GetChatHistory(userID string) ([]ChatMessage, error)
	SaveChatMessage(userID string, role string, content string) error
	ClearChatHistory(userID string) error
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Time    int64  `json:"time"`
}

type aiProcessor struct {
	apiKey string
	redis  *redis.Client
}

func NewAIProcessor(apiKey string, redisClient *redis.Client) AIProcessor {
	return &aiProcessor{
		apiKey: apiKey,
		redis:  redisClient,
	}
}

// ParseOrderMessage processes natural language order messages
func (a *aiProcessor) ParseOrderMessage(message string) (*models.Order, []models.OrderItem, error) {
	// Extract order information using regex patterns
	order := &models.Order{}
	var items []models.OrderItem
	
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
	order.Status = "pending"
	order.OrderDate = time.Now()
	
	return order, items, nil
}

// ExtractOrderItems extracts order items from natural language
func (a *aiProcessor) ExtractOrderItems(message string) ([]models.OrderItem, error) {
	var items []models.OrderItem
	
	// Pattern to match: "item name, qty X x price"
	itemRegex := regexp.MustCompile(`(?i)([a-zA-Z\s]+),\s*qty\s*(\d+)\s*x\s*(\d+(?:\.\d+)?)`)
	matches := itemRegex.FindAllStringSubmatch(message, -1)
	
	for _, match := range matches {
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

// ProcessWithOpenAI processes messages using OpenAI API with chat history
func (a *aiProcessor) ProcessWithOpenAI(message string, userID string) (string, interface{}, error) {
	if a.apiKey == "" || a.apiKey == "your_openai_api_key" {
		// Fallback to regex processing if no API key
		return a.ProcessWhatsAppMessage(message)
	}

	// Get chat history for context
	chatHistory, err := a.GetChatHistory(userID)
	if err != nil {
		// If error getting history, continue without context
		chatHistory = []ChatMessage{}
	}

	// Build messages array with system prompt and chat history
	messages := []map[string]string{
		{
			"role": "system",
			"content": `You are an AI assistant for a WhatsApp Task Management System. Analyze messages and return structured JSON responses.

MESSAGE TYPES TO DETECT:
1. add_user - "tambahkan user [username] [email] [phone] [role]", "/add_user"
2. create_order - "buat order [customer_name] [total_amount]", "/create_order" 
3. create_order_with_item - "buat order [customer] total [amount] item [item_name] [quantity] harga [price]"
4. assign_task - "assign task [title] [description] to [username]", "/assign_task"
5. view_tasks - "lihat tasks saya", "lihat task saya", "show my tasks", "show my task", "/my_tasks", "/my_daily_tasks", "/my_monthly_tasks"
6. view_orders - "lihat orders", "lihat order", "show orders", "show order", "list order", "list orders", "/view_orders"
7. list_users - "list user", "lihat users", "show users", "daftar user", "/list_users"
8. list_tasks - "/list_tasks"
9. add_order_item - "tambah item [order_id] [item_name] [quantity] [price] [description]"
10. view_order_items - "lihat items order [order_id]", "show order items [order_id]"
11. create_reminder - "buat reminder [task_id] [reminder_type] [scheduled_time]", "/create_reminder"
12. view_reminders - "lihat reminders", "lihat reminder", "show reminders", "show reminder", "/view_reminders"
13. update_progress - "/update_progress"
14. mark_complete - "/mark_complete"
15. my_report - "/my_report"
16. report_by_date - "/report_by_date"
17. clear_history - "/clear_history"
18. show_history - "/show_history"
19. help - "/help"
20. general - greetings, questions, general chat

RESPONSE FORMAT (JSON only):
{
  "type": "add_user|create_order|create_order_with_item|assign_task|view_tasks|view_orders|list_users|list_tasks|add_order_item|view_order_items|create_reminder|view_reminders|update_progress|mark_complete|my_report|report_by_date|clear_history|show_history|help|general",
  "data": {
    "username": "string",
    "email": "string", 
    "phone": "string",
    "role": "SuperAdmin|Admin|User",
    "customer_name": "string",
    "total_amount": "number",
    "title": "string",
    "description": "string",
    "assigned_to": "string",
    "order_id": "number",
    "item_name": "string",
    "quantity": "number",
    "price": "number",
    "task_id": "number",
    "reminder_type": "string",
    "scheduled_time": "string"
  },
  "message": "Friendly response message"
}

EXAMPLES:
Input: "tambahkan user ega egatryagung@gmail.com 08123456789 SuperAdmin"
Output: {"type":"add_user","data":{"username":"ega","email":"egatryagung@gmail.com","phone":"08123456789","role":"SuperAdmin"},"message":"I'll add user ega with SuperAdmin role"}

Input: "buat order John Doe 1000000"
Output: {"type":"create_order","data":{"customer_name":"John Doe","total_amount":1000000},"message":"I'll create an order for John Doe with total 1000000"}

Input: "buatkan order jhon total 10000 item ayam goreng 1 harga 10000"
Output: {"type":"create_order_with_item","data":{"customer_name":"jhon","total_amount":10000,"item_name":"ayam goreng","quantity":1,"price":10000},"message":"I'll create an order for jhon with ayam goreng item"}

Input: "list user"
Output: {"type":"list_users","data":{},"message":"I'll show you the list of users"}

Input: "list order"
Output: {"type":"view_orders","data":{},"message":"I'll show you the list of orders"}

Input: "lihat order"
Output: {"type":"view_orders","data":{},"message":"I'll show you the list of orders"}

Input: "/my_tasks"
Output: {"type":"view_tasks","data":{},"message":"I'll show you your tasks"}

Input: "/list_tasks"
Output: {"type":"list_tasks","data":{},"message":"I'll show you all tasks in the system"}

Input: "/update_progress"
Output: {"type":"update_progress","data":{},"message":"I'll help you update task progress"}

Input: "/mark_complete"
Output: {"type":"mark_complete","data":{},"message":"I'll help you mark task as complete"}

Input: "/help"
Output: {"type":"help","data":{},"message":"I'll show you available commands"}

Input: "tambah item 1 Laptop 2 5000000 Gaming laptop"
Output: {"type":"add_order_item","data":{"order_id":1,"item_name":"Laptop","quantity":2,"price":5000000,"description":"Gaming laptop"},"message":"I'll add 2 Laptop items to order 1"}

Input: "lihat items order 1"
Output: {"type":"view_order_items","data":{"order_id":1},"message":"I'll show you the items for order 1"}

Input: "buat reminder 1 deadline 2025-10-05 10:00"
Output: {"type":"create_reminder","data":{"task_id":1,"reminder_type":"deadline","scheduled_time":"2025-10-05 10:00"},"message":"I'll create a deadline reminder for task 1"}

Input: "lihat reminders"
Output: {"type":"view_reminders","data":{},"message":"I'll show you all reminders"}

Input: "halo"
Output: {"type":"general","data":{},"message":"Hello! How can I help you today?"}

Input: "unknown command"
Output: {"type":"general","data":{},"message":"I don't understand that command. Please use /help to see available commands."}

IMPORTANT: Always return valid JSON format only. No additional text.`,
		},
	}

	// Add chat history (in reverse order to maintain chronological order)
	for i := len(chatHistory) - 1; i >= 0; i-- {
		messages = append(messages, map[string]string{
			"role":    chatHistory[i].Role,
			"content": chatHistory[i].Content,
		})
	}

	// Add current message
	messages = append(messages, map[string]string{
		"role":    "user",
		"content": message,
	})

	// OpenAI API request
	requestBody := map[string]interface{}{
		"model":       "gpt-3.5-turbo",
		"messages":    messages,
		"max_tokens":  500,
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
	
	// Save user message and AI response to chat history
	a.SaveChatMessage(userID, "user", message)
	a.SaveChatMessage(userID, "assistant", content)
	
	// Try to determine if it's an order or task based on content
	if strings.Contains(strings.ToLower(content), "order") || strings.Contains(strings.ToLower(content), "total") {
		return "order", content, nil
	} else if strings.Contains(strings.ToLower(content), "task") || strings.Contains(strings.ToLower(content), "create") {
		return "task", content, nil
	}

	return "unknown", content, nil
}

// GetChatHistory retrieves the last 3 chat messages for a user
func (a *aiProcessor) GetChatHistory(userID string) ([]ChatMessage, error) {
	key := fmt.Sprintf("ai_chat_history:%s", userID)
	
	// Get all messages from Redis list
	messages, err := a.redis.LRange(key, 0, 2).Result()
	if err != nil {
		return nil, err
	}
	
	var chatHistory []ChatMessage
	for _, msg := range messages {
		var chatMsg ChatMessage
		if err := json.Unmarshal([]byte(msg), &chatMsg); err != nil {
			continue
		}
		chatHistory = append(chatHistory, chatMsg)
	}
	
	return chatHistory, nil
}

// SaveChatMessage saves a chat message to Redis
func (a *aiProcessor) SaveChatMessage(userID string, role string, content string) error {
	key := fmt.Sprintf("ai_chat_history:%s", userID)
	
	chatMsg := ChatMessage{
		Role:    role,
		Content: content,
		Time:    time.Now().Unix(),
	}
	
	msgJSON, err := json.Marshal(chatMsg)
	if err != nil {
		return err
	}
	
	// Add to the beginning of the list
	err = a.redis.LPush(key, msgJSON).Err()
	if err != nil {
		return err
	}
	
	// Keep only the last 3 messages
	err = a.redis.LTrim(key, 0, 2).Err()
	if err != nil {
		return err
	}
	
	// Set expiration to 10 minutes
	err = a.redis.Expire(key, 10*time.Minute).Err()
	if err != nil {
		return err
	}
	
	return nil
}

// ClearChatHistory clears chat history for a user
func (a *aiProcessor) ClearChatHistory(userID string) error {
	key := fmt.Sprintf("ai_chat_history:%s", userID)
	return a.redis.Del(key).Err()
}
