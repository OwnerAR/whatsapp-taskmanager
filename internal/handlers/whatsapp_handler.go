package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"task_manager/internal/models"
	"task_manager/internal/redis"
	"task_manager/internal/services"
	"time"

	"github.com/gin-gonic/gin"
)

type WhatsAppHandler struct {
	whatsappService services.WhatsAppService
	userService     services.UserService
	taskService     services.TaskService
	orderService    services.OrderService
	reminderService services.ReminderService
	aiProcessor     services.AIProcessor
}

// AIResponse represents structured AI response
type AIResponse struct {
	Type    string                 `json:"type"`
	Data    map[string]interface{} `json:"data"`
	Message string                 `json:"message"`
}

func NewWhatsAppHandler(
	whatsappService services.WhatsAppService,
	userService services.UserService,
	taskService services.TaskService,
	orderService services.OrderService,
	reminderService services.ReminderService,
	aiProcessor services.AIProcessor,
) *WhatsAppHandler {
	return &WhatsAppHandler{
		whatsappService: whatsappService,
		userService:     userService,
		taskService:     taskService,
		orderService:    orderService,
		reminderService: reminderService,
		aiProcessor:     aiProcessor,
	}
}

type WebhookRequest struct {
	SenderID  string `json:"sender_id"`
	ChatID    string `json:"chat_id"`
	From      string `json:"from"`
	Timestamp string `json:"timestamp"`
	Pushname  string `json:"pushname"`
	Message   struct {
		Text         string `json:"text"`
		ID           string `json:"id"`
		RepliedID    string `json:"replied_id"`
		QuotedMessage string `json:"quoted_message"`
	} `json:"message"`
}

type SendMessageRequest struct {
	Phone   string `json:"phone"`
	Message string `json:"message"`
}

func (h *WhatsAppHandler) HandleWebhook(c *gin.Context) {
	var req WebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Extract phone number from 'from' field (format: 628123456789@s.whatsapp.net)
	phoneNumber := req.From
	if phoneNumber == "" {
		phoneNumber = req.SenderID
	}
	
	// Remove @s.whatsapp.net suffix if present
	if strings.Contains(phoneNumber, "@s.whatsapp.net") {
		phoneNumber = strings.Replace(phoneNumber, "@s.whatsapp.net", "", 1)
	}

	// Get user by WhatsApp number
	user, err := h.userService.GetUserByWhatsAppNumber(phoneNumber)
	if err != nil {
		// Send error message
		h.whatsappService.SendMessage(phoneNumber, "❌ User not found. Please contact administrator.")
		c.JSON(http.StatusOK, gin.H{"status": "user_not_found"})
		return
	}

	// Process command
	response := h.processCommand(user, req.Message.Text)
	
	// Send response
	err = h.whatsappService.SendMessage(phoneNumber, response)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *WhatsAppHandler) SendMessage(c *gin.Context) {
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	err := h.whatsappService.SendMessage(req.Phone, req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *WhatsAppHandler) StartInteractiveSession(c *gin.Context) {
	var req struct {
		UserID      uint   `json:"user_id"`
		PhoneNumber string `json:"phone_number"`
		Command     string `json:"command"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	sessionID, err := h.whatsappService.StartInteractiveSession(req.UserID, req.PhoneNumber, req.Command)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"session_id": sessionID})
}

func (h *WhatsAppHandler) UpdateSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	
	var sessionData redis.SessionData
	if err := c.ShouldBindJSON(&sessionData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	err := h.whatsappService.UpdateSession(sessionID, &sessionData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *WhatsAppHandler) EndSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	
	err := h.whatsappService.EndSession(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to end session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *WhatsAppHandler) processCommand(user *models.User, message string) string {
	// Check if message is empty
	if strings.TrimSpace(message) == "" {
		return "❌ Empty message. Please send a message or use /help for available commands."
	}

	// AI-First Approach: Process all messages with AI, with /commands as fallback
	// Only use /commands for specific system operations like /help, /clear_history, etc.
	if strings.HasPrefix(strings.TrimSpace(message), "/") {
		// Parse command
		parts := strings.Fields(message)
		command := parts[0]
		args := parts[1:]
		
		// Only handle specific system commands directly
		switch command {
		case "/help":
			return h.getHelpMessage(user.Role)
		case "/clear_history":
			return h.clearChatHistory(user.ID)
		case "/show_history":
			return h.showChatHistory(user.ID)
		default:
			// For other /commands, try AI processing first
			return h.processAICommand(user, message)
		}
	} else {
		// Handle all natural language messages with AI
		return h.processAICommand(user, message)
	}
}

// processAICommand handles all messages with AI-first approach
func (h *WhatsAppHandler) processAICommand(user *models.User, message string) string {
	// Convert user ID to string for AI processor
	userID := fmt.Sprintf("%d", user.ID)
	
	// Process message with AI
	messageType, result, err := h.aiProcessor.ProcessWithOpenAI(message, userID)
	if err != nil {
		// Fallback to basic processing if AI fails
		return "🤖 I'm having trouble understanding your message. Please try using a command like /help for available options."
	}
	
	// Parse structured JSON response from AI
	aiResponse, err := h.parseAIResponse(result)
	if err != nil {
		// Fallback to general response if JSON parsing fails
		return fmt.Sprintf("🤖 %s", result)
	}
	
	// Handle different types of AI responses with actual database operations
	switch aiResponse.Type {
	case "add_user":
		return h.handleStructuredAIAddUser(user, aiResponse)
	case "create_order":
		return h.handleStructuredAICreateOrder(user, aiResponse)
	case "assign_task":
		return h.handleStructuredAIAssignTask(user, aiResponse)
	case "view_tasks":
		return h.handleAIViewTasks(user, message, result)
	case "view_orders":
		return h.handleAIViewOrders(user, message, result)
	case "general":
		// General AI response
		return fmt.Sprintf("🤖 %s", aiResponse.Message)
	default:
		// Try to detect intent and provide helpful response
		return h.handleAIGeneralIntent(user, message, result)
	}
}

// processNaturalLanguageMessage - kept for backward compatibility
func (h *WhatsAppHandler) processNaturalLanguageMessage(user *models.User, message string) string {
	return h.processAICommand(user, message)
}

// parseAIResponse parses structured JSON response from AI
func (h *WhatsAppHandler) parseAIResponse(result interface{}) (*AIResponse, error) {
	var aiResponse AIResponse
	
	// Convert result to string if needed
	var jsonStr string
	switch v := result.(type) {
	case string:
		jsonStr = v
	default:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}
		jsonStr = string(jsonBytes)
	}
	
	// Parse JSON
	err := json.Unmarshal([]byte(jsonStr), &aiResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}
	
	return &aiResponse, nil
}

// handleStructuredAIAddUser handles structured AI add user requests
func (h *WhatsAppHandler) handleStructuredAIAddUser(user *models.User, aiResponse *AIResponse) string {
	// Check if user has SuperAdmin access
	if user.Role != string(models.SuperAdmin) {
		return "❌ Anda tidak memiliki akses untuk menambah user. Hanya Super Admin yang dapat melakukan operasi ini."
	}
	
	// Extract data from AI response
	username, _ := aiResponse.Data["username"].(string)
	email, _ := aiResponse.Data["email"].(string)
	phone, _ := aiResponse.Data["phone"].(string)
	role, _ := aiResponse.Data["role"].(string)
	
	// Validate required fields
	if username == "" || email == "" || phone == "" || role == "" {
		return "❌ Data tidak lengkap. Pastikan username, email, phone, dan role tersedia."
	}
	
	// Validate role
	validRoles := []string{"SuperAdmin", "Admin", "User"}
	validRole := false
	for _, r := range validRoles {
		if strings.EqualFold(role, r) {
			role = r
			validRole = true
			break
		}
	}
	
	if !validRole {
		return "❌ Role tidak valid. Gunakan: SuperAdmin, Admin, atau User"
	}
	
	// Convert phone format if needed
	if strings.HasPrefix(phone, "08") {
		phone = "62" + phone[1:]
	}
	
	// Create user
	newUser := &models.User{
		Username:       username,
		Email:          email,
		PhoneNumber:    phone,
		WhatsAppNumber: phone,
		Role:           role,
		IsActive:       true,
	}
	
	err := h.userService.CreateUser(newUser, "default123")
	if err != nil {
		return fmt.Sprintf("❌ Gagal menambah user: %s", err.Error())
	}
	
	return fmt.Sprintf("✅ User berhasil ditambahkan!\n👤 Username: %s\n📧 Email: %s\n📱 Phone: %s\n🔑 Role: %s\n🔐 Password: default123", username, email, phone, role)
}

// handleStructuredAICreateOrder handles structured AI create order requests
func (h *WhatsAppHandler) handleStructuredAICreateOrder(user *models.User, aiResponse *AIResponse) string {
	// Check if user has Admin or SuperAdmin access
	if user.Role != string(models.Admin) && user.Role != string(models.SuperAdmin) {
		return "❌ Anda tidak memiliki akses untuk membuat order. Hanya Admin atau Super Admin yang dapat melakukan operasi ini."
	}
	
	// Extract data from AI response
	customerName, _ := aiResponse.Data["customer_name"].(string)
	totalAmountFloat, _ := aiResponse.Data["total_amount"].(float64)
	
	// Validate required fields
	if customerName == "" || totalAmountFloat == 0 {
		return "❌ Data tidak lengkap. Pastikan customer_name dan total_amount tersedia."
	}
	
	// Create order using existing service
	order := &models.Order{
		CustomerName: customerName,
		TotalAmount:  totalAmountFloat,
		Status:       "pending",
		OrderDate:    time.Now(),
		CreatedBy:    user.ID,
	}
	
	err := h.orderService.CreateOrder(order)
	if err != nil {
		return fmt.Sprintf("❌ Gagal membuat order: %s", err.Error())
	}
	
	return fmt.Sprintf("✅ Order berhasil dibuat!\n📦 Customer: %s\n💰 Total: Rp %.0f\n📅 Tanggal: %s", 
		customerName, totalAmountFloat, order.OrderDate.Format("2006-01-02 15:04"))
}

// handleStructuredAIAssignTask handles structured AI assign task requests
func (h *WhatsAppHandler) handleStructuredAIAssignTask(user *models.User, aiResponse *AIResponse) string {
	// Check if user has Admin or SuperAdmin access
	if user.Role != string(models.Admin) && user.Role != string(models.SuperAdmin) {
		return "❌ Anda tidak memiliki akses untuk menugaskan task. Hanya Admin atau Super Admin yang dapat melakukan operasi ini."
	}
	
	// Extract data from AI response
	title, _ := aiResponse.Data["title"].(string)
	description, _ := aiResponse.Data["description"].(string)
	assignedToUsername, _ := aiResponse.Data["assigned_to"].(string)
	
	// Validate required fields
	if title == "" || description == "" || assignedToUsername == "" {
		return "❌ Data tidak lengkap. Pastikan title, description, dan assigned_to tersedia."
	}
	
	// Find user by username
	assignedUser, err := h.userService.GetUserByUsername(assignedToUsername)
	if err != nil {
		return fmt.Sprintf("❌ User '%s' tidak ditemukan. Pastikan username benar.", assignedToUsername)
	}
	
	// Create task
	task := &models.Task{
		Title:       title,
		Description: description,
		AssignedTo:  assignedUser.ID,
		Status:      string(models.Pending),
		Priority:    string(models.Medium),
		TaskType:    string(models.Custom),
		CreatedBy:   user.ID,
	}
	
	err = h.taskService.CreateTask(task)
	if err != nil {
		return fmt.Sprintf("❌ Gagal membuat task: %s", err.Error())
	}
	
	return fmt.Sprintf("✅ Task berhasil ditugaskan!\n📝 Title: %s\n📄 Description: %s\n👤 Assigned to: %s", 
		title, description, assignedToUsername)
}

// handleAIAddUser processes AI-detected add user requests
func (h *WhatsAppHandler) handleAIAddUser(user *models.User, message string, aiResult interface{}) string {
	// Check if user has SuperAdmin access
	if user.Role != string(models.SuperAdmin) {
		return "❌ Anda tidak memiliki akses untuk menambah user. Hanya Super Admin yang dapat melakukan operasi ini."
	}
	
	// Parse user information from message using regex
	// Pattern: "tambahkan user [username] [email] [phone] [role]"
	userRegex := regexp.MustCompile(`(?i)(?:tambahkan|add|create)\s+user\s+(\w+)\s+([^\s]+@[^\s]+)\s+(\d+)\s+(\w+)`)
	matches := userRegex.FindStringSubmatch(message)
	
	if len(matches) < 5 {
		return "❌ Format tidak valid. Gunakan: 'tambahkan user [username] [email] [phone] [role]'\nContoh: 'tambahkan user ega egatryagung@gmail.com 08123456789 SuperAdmin'"
	}
	
	username := matches[1]
	email := matches[2]
	phone := matches[3]
	role := matches[4]
	
	// Validate role
	validRoles := []string{"SuperAdmin", "Admin", "User"}
	validRole := false
	for _, r := range validRoles {
		if strings.EqualFold(role, r) {
			role = r
			validRole = true
			break
		}
	}
	
	if !validRole {
		return "❌ Role tidak valid. Gunakan: SuperAdmin, Admin, atau User"
	}
	
	// Convert phone format if needed
	if strings.HasPrefix(phone, "08") {
		phone = "62" + phone[1:]
	}
	
	// Create user
	newUser := &models.User{
		Username:       username,
		Email:          email,
		PhoneNumber:    phone,
		WhatsAppNumber: phone,
		Role:           role,
		IsActive:       true,
	}
	
	err := h.userService.CreateUser(newUser, "default123")
	if err != nil {
		return fmt.Sprintf("❌ Gagal menambah user: %s", err.Error())
	}
	
	return fmt.Sprintf("✅ User berhasil ditambahkan!\n👤 Username: %s\n📧 Email: %s\n📱 Phone: %s\n🔑 Role: %s\n🔐 Password: default123", username, email, phone, role)
}

// handleAICreateOrder processes AI-detected create order requests
func (h *WhatsAppHandler) handleAICreateOrder(user *models.User, message string, aiResult interface{}) string {
	// Check if user has Admin or SuperAdmin access
	if user.Role != string(models.Admin) && user.Role != string(models.SuperAdmin) {
		return "❌ Anda tidak memiliki akses untuk membuat order. Hanya Admin atau Super Admin yang dapat melakukan operasi ini."
	}
	
	// Parse order information from message
	orderRegex := regexp.MustCompile(`(?i)(?:buat|create|tambah)\s+order\s+([^0-9]+)\s+(\d+(?:\.\d+)?)`)
	matches := orderRegex.FindStringSubmatch(message)
	
	if len(matches) < 3 {
		return "❌ Format tidak valid. Gunakan: 'buat order [customer_name] [total_amount]'\nContoh: 'buat order John Doe 1000000'"
	}
	
	customerName := strings.TrimSpace(matches[1])
	totalAmountStr := matches[2]
	
	totalAmount, err := strconv.ParseFloat(totalAmountStr, 64)
	if err != nil {
		return "❌ Total amount tidak valid. Gunakan angka yang benar."
	}
	
	// Create order using existing service
	order := &models.Order{
		CustomerName: customerName,
		TotalAmount:  totalAmount,
		Status:       "pending",
		OrderDate:    time.Now(),
		CreatedBy:    user.ID,
	}
	
	err = h.orderService.CreateOrder(order)
	if err != nil {
		return fmt.Sprintf("❌ Gagal membuat order: %s", err.Error())
	}
	
	return fmt.Sprintf("✅ Order berhasil dibuat!\n📦 Customer: %s\n💰 Total: Rp %.0f\n📅 Tanggal: %s", 
		customerName, totalAmount, order.OrderDate.Format("2006-01-02 15:04"))
}

// handleAIAssignTask processes AI-detected assign task requests
func (h *WhatsAppHandler) handleAIAssignTask(user *models.User, message string, aiResult interface{}) string {
	// Check if user has Admin or SuperAdmin access
	if user.Role != string(models.Admin) && user.Role != string(models.SuperAdmin) {
		return "❌ Anda tidak memiliki akses untuk menugaskan task. Hanya Admin atau Super Admin yang dapat melakukan operasi ini."
	}
	
	// Parse task information from message
	taskRegex := regexp.MustCompile(`(?i)(?:assign|tugaskan|berikan)\s+task\s+(\w+)\s+(.+?)\s+to\s+(\w+)`)
	matches := taskRegex.FindStringSubmatch(message)
	
	if len(matches) < 4 {
		return "❌ Format tidak valid. Gunakan: 'assign task [title] [description] to [username]'\nContoh: 'assign task Update Website Update homepage design to john'"
	}
	
	title := strings.TrimSpace(matches[1])
	description := strings.TrimSpace(matches[2])
	assignedToUsername := strings.TrimSpace(matches[3])
	
	// Find user by username
	assignedUser, err := h.userService.GetUserByUsername(assignedToUsername)
	if err != nil {
		return fmt.Sprintf("❌ User '%s' tidak ditemukan. Pastikan username benar.", assignedToUsername)
	}
	
	// Create task
	task := &models.Task{
		Title:       title,
		Description: description,
		AssignedTo:  assignedUser.ID,
		Status:      string(models.Pending),
		Priority:    string(models.Medium),
		TaskType:    string(models.Custom),
		CreatedBy:   user.ID,
	}
	
	err = h.taskService.CreateTask(task)
	if err != nil {
		return fmt.Sprintf("❌ Gagal membuat task: %s", err.Error())
	}
	
	return fmt.Sprintf("✅ Task berhasil ditugaskan!\n📝 Title: %s\n📄 Description: %s\n👤 Assigned to: %s", 
		title, description, assignedToUsername)
}

// handleAIViewTasks processes AI-detected view tasks requests
func (h *WhatsAppHandler) handleAIViewTasks(user *models.User, message string, aiResult interface{}) string {
	tasks, err := h.taskService.GetTasksByUser(user.ID)
	if err != nil {
		return fmt.Sprintf("❌ Gagal mengambil tasks: %s", err.Error())
	}
	
	if len(tasks) == 0 {
		return "📝 Tidak ada task yang ditugaskan kepada Anda."
	}
	
	response := "📝 **Your Tasks:**\n\n"
	for _, task := range tasks {
		status := "❌ Pending"
		if task.Status == string(models.InProgress) {
			status = "🔄 In Progress"
		} else if task.Status == string(models.Completed) {
			status = "✅ Completed"
		}
		
		response += fmt.Sprintf("**%s**\n", task.Title)
		response += fmt.Sprintf("Description: %s\n", task.Description)
		response += fmt.Sprintf("Status: %s\n", status)
		response += fmt.Sprintf("Progress: %d%%\n\n", task.CompletionPercentage)
	}
	
	return response
}

// handleAIViewOrders processes AI-detected view orders requests
func (h *WhatsAppHandler) handleAIViewOrders(user *models.User, message string, aiResult interface{}) string {
	// Check if user has Admin or SuperAdmin access for all orders
	if user.Role == string(models.Admin) || user.Role == string(models.SuperAdmin) {
		orders, err := h.orderService.GetAllOrders()
		if err != nil {
			return fmt.Sprintf("❌ Gagal mengambil orders: %s", err.Error())
		}
		
		if len(orders) == 0 {
			return "📦 Tidak ada order yang ditemukan."
		}
		
		response := "📦 **All Orders:**\n\n"
		for _, order := range orders {
			response += fmt.Sprintf("**Order #%d**\n", order.ID)
			response += fmt.Sprintf("Customer: %s\n", order.CustomerName)
			response += fmt.Sprintf("Total: Rp %.0f\n", order.TotalAmount)
			response += fmt.Sprintf("Status: %s\n\n", order.Status)
		}
		
		return response
	} else {
		// Regular users can only see their own orders
		orders, err := h.orderService.GetOrdersByUser(user.ID)
		if err != nil {
			return fmt.Sprintf("❌ Gagal mengambil orders: %s", err.Error())
		}
		
		if len(orders) == 0 {
			return "📦 Tidak ada order yang terkait dengan Anda."
		}
		
		response := "📦 **Your Orders:**\n\n"
		for _, order := range orders {
			response += fmt.Sprintf("**Order #%d**\n", order.ID)
			response += fmt.Sprintf("Customer: %s\n", order.CustomerName)
			response += fmt.Sprintf("Total: Rp %.0f\n", order.TotalAmount)
			response += fmt.Sprintf("Status: %s\n\n", order.Status)
		}
		
		return response
	}
}

// handleAIGeneralIntent handles general AI responses
func (h *WhatsAppHandler) handleAIGeneralIntent(user *models.User, message string, aiResult interface{}) string {
	// Check for common intents and provide helpful responses
	messageLower := strings.ToLower(message)
	
	if strings.Contains(messageLower, "halo") || strings.Contains(messageLower, "hi") || strings.Contains(messageLower, "hello") {
		return fmt.Sprintf("👋 Halo %s! Saya AI assistant untuk Task Manager.\n\nSaya dapat membantu Anda dengan:\n• Menambah user (Super Admin)\n• Membuat order (Admin)\n• Menugaskan task (Admin)\n• Melihat tasks dan orders\n\nCoba katakan: 'lihat tasks saya' atau 'buat order John 1000000'", user.Username)
	}
	
	if strings.Contains(messageLower, "help") || strings.Contains(messageLower, "bantuan") {
		return h.getHelpMessage(user.Role)
	}
	
	// Default AI response
	return fmt.Sprintf("🤖 %s", aiResult)
}

func (h *WhatsAppHandler) processAdminCommand(user *models.User, command string, args []string) string {
	switch command {
	case "/add_user":
		return h.addUser(user, args)
	case "/list_users":
		return h.listUsers()
	case "/list_tasks":
		return h.listAllTasks()
	case "/create_order":
		return h.createOrder(user.ID, args)
	case "/view_orders":
		return h.getAllOrders()
	case "/assign_task":
		return h.assignTask(user.ID, args)
	case "/create_daily_task":
		return h.createDailyTask(user.ID, args)
	case "/create_monthly_task":
		return h.createMonthlyTask(user.ID, args)
	case "/set_tax_rate":
		return h.setTaxRate(user.ID, args)
	case "/set_marketing_rate":
		return h.setMarketingRate(user.ID, args)
	case "/set_rental_rate":
		return h.setRentalRate(user.ID, args)
	case "/generate_report":
		return h.generateReport()
	case "/daily_report":
		return h.generateDailyReport()
	case "/monthly_report":
		return h.generateMonthlyReport()
	default:
		return "❌ Unknown admin command. Type /help for available commands."
	}
}

func (h *WhatsAppHandler) getHelpMessage(role string) string {
	baseCommands := `
📱 **Available Commands:**

**General Commands:**
/my_tasks - View assigned tasks
/my_daily_tasks - View today's daily tasks
/my_monthly_tasks - View this month's tasks
/update_progress [task_id] [percentage] - Update task progress
/mark_complete [task_id] - Mark task as implemented
/view_orders - View related orders
/my_report - View personal financial reports
/report_by_date [start_date] [end_date] - Generate reports by date range
/clear_history - Clear AI chat history
/show_history - Show AI chat history
/help - Show this help message
`

	if role == string(models.Admin) {
		baseCommands += `
**Admin Commands:**
/create_order [customer_name] [total_amount] - Create new order
/view_orders - List all orders
/assign_task [username_or_id] [title] [description] - Assign task to user
/create_daily_task [username_or_id] [title] [description] - Create daily recurring task
/create_monthly_task [username_or_id] [title] [description] - Create monthly recurring task
/set_tax_rate [percentage] - Set tax percentage
/set_marketing_rate [percentage] - Set marketing cost percentage
/set_rental_rate [percentage] - Set rental cost percentage
/generate_report - Generate financial reports
/daily_report - Generate daily report
/monthly_report - Generate monthly report
`
	}

	if role == string(models.SuperAdmin) {
		baseCommands += `
**Super Admin Commands:**
/add_user [username] [email] [phone] [role] - Add new user
/list_users - View all users (shows User ID for reference)
/list_tasks - View all tasks in the system
/update_user - Update user information
/delete_user - Delete user
/set_role - Change user role
/system_config - System configuration

**Admin Commands:**
/create_order [customer_name] [total_amount] - Create new order
/view_orders - List all orders
/assign_task [username_or_id] [title] [description] - Assign task to user
/create_daily_task [username_or_id] [title] [description] - Create daily recurring task
/create_monthly_task [username_or_id] [title] [description] - Create monthly recurring task
/set_tax_rate [percentage] - Set tax percentage
/set_marketing_rate [percentage] - Set marketing cost percentage
/set_rental_rate [percentage] - Set rental cost percentage
/generate_report - Generate financial reports
/daily_report - Generate daily report
/monthly_report - Generate monthly report
`
	}

	return baseCommands
}

func (h *WhatsAppHandler) clearChatHistory(userID uint) string {
	// Clear chat history for AI memory
	err := h.aiProcessor.ClearChatHistory(fmt.Sprintf("%d", userID))
	if err != nil {
		return "❌ Failed to clear chat history: " + err.Error()
	}
	return "✅ Chat history cleared successfully"
}

func (h *WhatsAppHandler) showChatHistory(userID uint) string {
	// Show chat history for AI memory
	history, err := h.aiProcessor.GetChatHistory(fmt.Sprintf("%d", userID))
	if err != nil {
		return "❌ Failed to get chat history: " + err.Error()
	}
	
	if len(history) == 0 {
		return "📝 **Chat History:**\n\nNo chat history found."
	}
	
	response := "📝 **Chat History (Last 3 messages, expires in 10 minutes):**\n\n"
	for i, msg := range history {
		role := "👤 User"
		if msg.Role == "assistant" {
			role = "🤖 AI"
		}
		response += fmt.Sprintf("%d. %s: %s\n", i+1, role, msg.Content)
		response += fmt.Sprintf("   Time: %s\n\n", time.Unix(msg.Time, 0).Format("2006-01-02 15:04:05"))
	}
	
	return response
}

func (h *WhatsAppHandler) getUserTasks(userID uint) string {
	tasks, err := h.taskService.GetTasksByUser(userID)
	if err != nil {
		return "❌ Failed to get tasks: " + err.Error()
	}

	if len(tasks) == 0 {
		return "📝 No tasks assigned to you."
	}

	response := "📝 **Your Tasks:**\n\n"
	for _, task := range tasks {
		status := "⏳ Pending"
		if task.Status == string(models.InProgress) {
			status = "🔄 In Progress"
		} else if task.Status == string(models.Completed) {
			status = "✅ Completed"
		}

		response += fmt.Sprintf("**%s**\n", task.Title)
		response += fmt.Sprintf("Status: %s\n", status)
		response += fmt.Sprintf("Progress: %d%%\n", task.CompletionPercentage)
		response += fmt.Sprintf("Priority: %s\n", task.Priority)
		if task.DueDate != nil {
			response += fmt.Sprintf("Due: %s\n", task.DueDate.Format("2006-01-02"))
		}
		response += "\n"
	}

	return response
}

func (h *WhatsAppHandler) getDailyTasks(userID uint) string {
	tasks, err := h.taskService.GetDailyTasks(userID, time.Now())
	if err != nil {
		return "❌ Failed to get daily tasks: " + err.Error()
	}

	if len(tasks) == 0 {
		return "📅 No daily tasks for today."
	}

	response := "📅 **Today's Daily Tasks:**\n\n"
	for _, task := range tasks {
		response += fmt.Sprintf("**%s**\n", task.Title)
		response += fmt.Sprintf("Progress: %d%%\n", task.CompletionPercentage)
		response += fmt.Sprintf("Implemented: %t\n", task.IsImplemented)
		response += "\n"
	}

	return response
}

func (h *WhatsAppHandler) getMonthlyTasks(userID uint) string {
	monthYear := time.Now().Format("2006-01")
	tasks, err := h.taskService.GetMonthlyTasks(userID, monthYear)
	if err != nil {
		return "❌ Failed to get monthly tasks: " + err.Error()
	}

	if len(tasks) == 0 {
		return "📅 No monthly tasks for this month."
	}

	response := "📅 **This Month's Tasks:**\n\n"
	for _, task := range tasks {
		response += fmt.Sprintf("**%s**\n", task.Title)
		response += fmt.Sprintf("Progress: %d%%\n", task.CompletionPercentage)
		response += fmt.Sprintf("Implemented: %t\n", task.IsImplemented)
		response += "\n"
	}

	return response
}

func (h *WhatsAppHandler) updateTaskProgress(userID uint, args []string) string {
	if len(args) < 2 {
		return "❌ Usage: /update_progress [task_id] [percentage]"
	}

	taskID, err := strconv.ParseUint(args[0], 10, 32)
	if err != nil {
		return "❌ Invalid task ID"
	}

	progress, err := strconv.Atoi(args[1])
	if err != nil || progress < 0 || progress > 100 {
		return "❌ Invalid progress percentage (0-100)"
	}

	err = h.taskService.UpdateTaskProgress(uint(taskID), progress, false, "", userID)
	if err != nil {
		return "❌ Failed to update progress: " + err.Error()
	}

	return fmt.Sprintf("✅ Task progress updated to %d%%", progress)
}

func (h *WhatsAppHandler) markTaskComplete(userID uint, args []string) string {
	if len(args) < 1 {
		return "❌ Usage: /mark_complete [task_id]"
	}

	taskID, err := strconv.ParseUint(args[0], 10, 32)
	if err != nil {
		return "❌ Invalid task ID"
	}

	err = h.taskService.UpdateTaskProgress(uint(taskID), 100, true, "Task completed", userID)
	if err != nil {
		return "❌ Failed to mark task as complete: " + err.Error()
	}

	return "✅ Task marked as implemented"
}

func (h *WhatsAppHandler) getUserOrders(userID uint) string {
	orders, err := h.orderService.GetOrdersByUser(userID)
	if err != nil {
		return "❌ Failed to get orders: " + err.Error()
	}

	if len(orders) == 0 {
		return "📦 No orders found."
	}

	response := "📦 **Your Orders:**\n\n"
	for _, order := range orders {
		response += fmt.Sprintf("**Order #%s**\n", order.OrderNumber)
		response += fmt.Sprintf("Customer: %s\n", order.CustomerName)
		response += fmt.Sprintf("Total: $%.2f\n", order.TotalAmount)
		response += fmt.Sprintf("Status: %s\n", order.Status)
		response += fmt.Sprintf("Date: %s\n", order.OrderDate.Format("2006-01-02"))
		response += "\n"
	}

	return response
}

func (h *WhatsAppHandler) getUserReport(userID uint) string {
	// Implementation for user report
	return "📊 **Your Personal Report:**\n\nThis feature will show your personal financial summary."
}

func (h *WhatsAppHandler) getReportByDate(userID uint, args []string) string {
	if len(args) < 2 {
		return "❌ Usage: /report_by_date [start_date] [end_date] (format: YYYY-MM-DD)"
	}

	startDate, err := time.Parse("2006-01-02", args[0])
	if err != nil {
		return "❌ Invalid start date format. Use YYYY-MM-DD"
	}

	endDate, err := time.Parse("2006-01-02", args[1])
	if err != nil {
		return "❌ Invalid end date format. Use YYYY-MM-DD"
	}

	orders, err := h.orderService.GetOrdersByDateRange(startDate, endDate)
	if err != nil {
		return "❌ Failed to get orders: " + err.Error()
	}

	if len(orders) == 0 {
		return "📊 No orders found for the specified date range."
	}

	totalAmount := 0.0
	for _, order := range orders {
		totalAmount += order.TotalAmount
	}

	response := fmt.Sprintf("📊 **Report for %s to %s:**\n\n", args[0], args[1])
	response += fmt.Sprintf("Total Orders: %d\n", len(orders))
	response += fmt.Sprintf("Total Amount: $%.2f\n", totalAmount)

	return response
}

// Admin command implementations
func (h *WhatsAppHandler) addUser(user *models.User, args []string) string {
	if len(args) < 4 {
		return "❌ Usage: /add_user [username] [email] [phone] [role]"
	}

	newUser := &models.User{
		Username:       args[0],
		Email:          args[1],
		PhoneNumber:    args[2],
		Role:           args[3],
		WhatsAppNumber: args[2],
		IsActive:       true,
	}

	err := h.userService.CreateUser(newUser, "default_password")
	if err != nil {
		return "❌ Failed to create user: " + err.Error()
	}

	return "✅ User created successfully"
}

func (h *WhatsAppHandler) listUsers() string {
	users, err := h.userService.GetAllUsers()
	if err != nil {
		return "❌ Failed to get users: " + err.Error()
	}

	response := "👥 **All Users:**\n\n"
	for _, user := range users {
		status := "❌ Inactive"
		if user.IsActive {
			status = "✅ Active"
		}
		response += fmt.Sprintf("**ID: %d** - **%s** (%s)\n", user.ID, user.Username, user.Email)
		response += fmt.Sprintf("Role: %s\n", user.Role)
		response += fmt.Sprintf("Status: %s\n", status)
		response += "\n"
	}

	return response
}

func (h *WhatsAppHandler) listAllTasks() string {
	tasks, err := h.taskService.GetAllTasks()
	if err != nil {
		return "❌ Failed to get tasks: " + err.Error()
	}

	if len(tasks) == 0 {
		return "📝 **All Tasks:**\n\nNo tasks found."
	}

	response := "📝 **All Tasks:**\n\n"
	for _, task := range tasks {
		status := "❌ Pending"
		if task.Status == string(models.InProgress) {
			status = "🔄 In Progress"
		} else if task.Status == string(models.Completed) {
			status = "✅ Completed"
		} else if task.Status == string(models.Overdue) {
			status = "⚠️ Overdue"
		}

		priority := "🟡 Medium"
		if task.Priority == string(models.High) {
			priority = "🔴 High"
		} else if task.Priority == string(models.Low) {
			priority = "🟢 Low"
		} else if task.Priority == string(models.Urgent) {
			priority = "🚨 Urgent"
		}

		implemented := "❌ Not Implemented"
		if task.IsImplemented {
			implemented = "✅ Implemented"
		}

		response += fmt.Sprintf("**ID: %d** - **%s**\n", task.ID, task.Title)
		response += fmt.Sprintf("Description: %s\n", task.Description)
		response += fmt.Sprintf("Assigned To: User ID %d\n", task.AssignedTo)
		response += fmt.Sprintf("Status: %s\n", status)
		response += fmt.Sprintf("Priority: %s\n", priority)
		response += fmt.Sprintf("Progress: %d%%\n", task.CompletionPercentage)
		response += fmt.Sprintf("Implemented: %s\n", implemented)
		if task.DueDate != nil {
			response += fmt.Sprintf("Due Date: %s\n", task.DueDate.Format("2006-01-02 15:04"))
		}
		if task.CompletedAt != nil {
			response += fmt.Sprintf("Completed: %s\n", task.CompletedAt.Format("2006-01-02 15:04"))
		}
		response += "\n"
	}

	return response
}

func (h *WhatsAppHandler) createOrder(userID uint, args []string) string {
	if len(args) < 2 {
		return "❌ Usage: /create_order [customer_name] [total_amount]"
	}

	totalAmount, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return "❌ Invalid total amount"
	}

	order := &models.Order{
		OrderNumber:  fmt.Sprintf("ORD-%d", time.Now().Unix()),
		CustomerName: args[0],
		TotalAmount:  totalAmount,
		Status:       string(models.OrderPending),
		OrderDate:    time.Now(),
		CreatedBy:    userID,
	}

	err = h.orderService.CreateOrder(order)
	if err != nil {
		return "❌ Failed to create order: " + err.Error()
	}

	return fmt.Sprintf("✅ Order created successfully\nOrder #: %s\nCustomer: %s\nTotal: $%.2f", 
		order.OrderNumber, order.CustomerName, order.TotalAmount)
}

func (h *WhatsAppHandler) getAllOrders() string {
	orders, err := h.orderService.GetAllOrders()
	if err != nil {
		return "❌ Failed to get orders: " + err.Error()
	}

	if len(orders) == 0 {
		return "📦 No orders found."
	}

	response := "📦 **All Orders:**\n\n"
	for _, order := range orders {
		response += fmt.Sprintf("**Order #%s**\n", order.OrderNumber)
		response += fmt.Sprintf("Customer: %s\n", order.CustomerName)
		response += fmt.Sprintf("Total: $%.2f\n", order.TotalAmount)
		response += fmt.Sprintf("Status: %s\n", order.Status)
		response += fmt.Sprintf("Date: %s\n", order.OrderDate.Format("2006-01-02"))
		response += "\n"
	}

	return response
}

func (h *WhatsAppHandler) assignTask(userID uint, args []string) string {
	if len(args) < 3 {
		return "❌ Usage: /assign_task [username_or_id] [title] [description]"
	}

	// Try to parse as user ID first
	var assignedTo uint
	if userID, err := strconv.ParseUint(args[0], 10, 32); err == nil {
		assignedTo = uint(userID)
	} else {
		// If not a number, treat as username
		user, err := h.userService.GetUserByUsername(args[0])
			if err != nil {
				return "❌ User not found: " + args[0]
			}
			assignedTo = user.ID
		}

		// Join all args after title as description
		description := strings.Join(args[2:], " ")
		
		task := &models.Task{
			Title:       args[1],
			Description: description,
			AssignedTo:  uint(assignedTo),
		Status:      string(models.Pending),
		Priority:    string(models.Medium),
		TaskType:    string(models.Custom),
		CreatedBy:   userID,
	}

	err := h.taskService.CreateTask(task)
	if err != nil {
		return "❌ Failed to create task: " + err.Error()
	}

	return "✅ Task assigned successfully"
}

func (h *WhatsAppHandler) createDailyTask(userID uint, args []string) string {
	if len(args) < 3 {
		return "❌ Usage: /create_daily_task [username_or_id] [title] [description]"
	}

	// Try to parse as user ID first
	var assignedTo uint
	if userID, err := strconv.ParseUint(args[0], 10, 32); err == nil {
		assignedTo = uint(userID)
	} else {
		// If not a number, treat as username
		user, err := h.userService.GetUserByUsername(args[0])
			if err != nil {
				return "❌ User not found: " + args[0]
			}
			assignedTo = user.ID
		}

		// Join all args after title as description
		description := strings.Join(args[2:], " ")
		
		task := &models.Task{
			Title:       args[1],
			Description: description,
			AssignedTo:  uint(assignedTo),
		Status:      string(models.Pending),
		Priority:    string(models.Medium),
		CreatedBy:   userID,
	}

	err := h.taskService.CreateDailyTask(task)
	if err != nil {
		return "❌ Failed to create daily task: " + err.Error()
	}

	return "✅ Daily task created successfully"
}

func (h *WhatsAppHandler) createMonthlyTask(userID uint, args []string) string {
	if len(args) < 3 {
		return "❌ Usage: /create_monthly_task [username_or_id] [title] [description]"
	}

	// Try to parse as user ID first
	var assignedTo uint
	if userID, err := strconv.ParseUint(args[0], 10, 32); err == nil {
		assignedTo = uint(userID)
	} else {
		// If not a number, treat as username
		user, err := h.userService.GetUserByUsername(args[0])
			if err != nil {
				return "❌ User not found: " + args[0]
			}
			assignedTo = user.ID
		}

		// Join all args after title as description
		description := strings.Join(args[2:], " ")
		
		task := &models.Task{
			Title:       args[1],
			Description: description,
			AssignedTo:  uint(assignedTo),
		Status:      string(models.Pending),
		Priority:    string(models.Medium),
		CreatedBy:   userID,
	}

	err := h.taskService.CreateMonthlyTask(task)
	if err != nil {
		return "❌ Failed to create monthly task: " + err.Error()
	}

	return "✅ Monthly task created successfully"
}

func (h *WhatsAppHandler) setTaxRate(userID uint, args []string) string {
	if len(args) < 1 {
		return "❌ Usage: /set_tax_rate [percentage]"
	}

	percentage, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return "❌ Invalid percentage"
	}

	// Implementation for setting tax rate
	return fmt.Sprintf("✅ Tax rate set to %.2f%%", percentage)
}

func (h *WhatsAppHandler) setMarketingRate(userID uint, args []string) string {
	if len(args) < 1 {
		return "❌ Usage: /set_marketing_rate [percentage]"
	}

	percentage, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return "❌ Invalid percentage"
	}

	// Implementation for setting marketing rate
	return fmt.Sprintf("✅ Marketing rate set to %.2f%%", percentage)
}

func (h *WhatsAppHandler) setRentalRate(userID uint, args []string) string {
	if len(args) < 1 {
		return "❌ Usage: /set_rental_rate [percentage]"
	}

	percentage, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return "❌ Invalid percentage"
	}

	// Implementation for setting rental rate
	return fmt.Sprintf("✅ Rental rate set to %.2f%%", percentage)
}

func (h *WhatsAppHandler) generateReport() string {
	return "📊 **Financial Report:**\n\nThis feature will show comprehensive financial summary."
}

func (h *WhatsAppHandler) generateDailyReport() string {
	return "📊 **Daily Report:**\n\nThis feature will show today's financial summary."
}

func (h *WhatsAppHandler) generateMonthlyReport() string {
	return "📊 **Monthly Report:**\n\nThis feature will show this month's financial summary."
}