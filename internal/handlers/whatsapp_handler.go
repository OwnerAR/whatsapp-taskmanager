package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
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
}

func NewWhatsAppHandler(
	whatsappService services.WhatsAppService,
	userService services.UserService,
	taskService services.TaskService,
	orderService services.OrderService,
	reminderService services.ReminderService,
) *WhatsAppHandler {
	return &WhatsAppHandler{
		whatsappService: whatsappService,
		userService:     userService,
		taskService:     taskService,
		orderService:    orderService,
		reminderService: reminderService,
	}
}

type WebhookRequest struct {
	Phone   string `json:"phone"`
	Message string `json:"message"`
	From    string `json:"from"`
	To      string `json:"to"`
	Time    string `json:"time"`
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

	// Get user by WhatsApp number
	user, err := h.userService.GetUserByWhatsAppNumber(req.Phone)
	if err != nil {
		// Send error message
		h.whatsappService.SendMessage(req.Phone, "❌ User not found. Please contact administrator.")
		c.JSON(http.StatusOK, gin.H{"status": "user_not_found"})
		return
	}

	// Process command
	response := h.processCommand(user, req.Message)
	
	// Send response
	err = h.whatsappService.SendMessage(req.Phone, response)
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
	// Parse command
	parts := strings.Fields(message)
	if len(parts) == 0 {
		return "❌ Invalid command. Type /help for available commands."
	}

	command := parts[0]
	args := parts[1:]

	switch command {
	case "/help":
		return h.getHelpMessage(user.Role)
	case "/my_tasks":
		return h.getUserTasks(user.ID)
	case "/my_daily_tasks":
		return h.getDailyTasks(user.ID)
	case "/my_monthly_tasks":
		return h.getMonthlyTasks(user.ID)
	case "/update_progress":
		return h.updateTaskProgress(user.ID, args)
	case "/mark_complete":
		return h.markTaskComplete(user.ID, args)
	case "/view_orders":
		return h.getUserOrders(user.ID)
	case "/my_report":
		return h.getUserReport(user.ID)
	case "/report_by_date":
		return h.getReportByDate(user.ID, args)
	default:
		// Check if user is admin or super admin for admin commands
		if user.Role == string(models.Admin) || user.Role == string(models.SuperAdmin) {
			return h.processAdminCommand(user, command, args)
		}
		return "❌ Unknown command. Type /help for available commands."
	}
}

func (h *WhatsAppHandler) processAdminCommand(user *models.User, command string, args []string) string {
	switch command {
	case "/add_user":
		return h.addUser(user, args)
	case "/list_users":
		return h.listUsers()
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
/help - Show this help message
`

	if role == string(models.Admin) || role == string(models.SuperAdmin) {
		baseCommands += `
**Admin Commands:**
/add_user [username] [email] [phone] [role] - Add new user
/list_users - View all users
/create_order [customer_name] [total_amount] - Create new order
/view_orders - List all orders
/assign_task [user_id] [title] [description] - Assign task to user
/create_daily_task [user_id] [title] [description] - Create daily recurring task
/create_monthly_task [user_id] [title] [description] - Create monthly recurring task
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
		response += fmt.Sprintf("**%s** (%s)\n", user.Username, user.Email)
		response += fmt.Sprintf("Role: %s\n", user.Role)
		response += fmt.Sprintf("Status: %s\n", status)
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
		return "❌ Usage: /assign_task [user_id] [title] [description]"
	}

	assignedTo, err := strconv.ParseUint(args[0], 10, 32)
	if err != nil {
		return "❌ Invalid user ID"
	}

	task := &models.Task{
		Title:       args[1],
		Description: args[2],
		AssignedTo:  uint(assignedTo),
		Status:      string(models.Pending),
		Priority:    string(models.Medium),
		TaskType:    string(models.Custom),
		CreatedBy:   userID,
	}

	err = h.taskService.CreateTask(task)
	if err != nil {
		return "❌ Failed to create task: " + err.Error()
	}

	return "✅ Task assigned successfully"
}

func (h *WhatsAppHandler) createDailyTask(userID uint, args []string) string {
	if len(args) < 3 {
		return "❌ Usage: /create_daily_task [user_id] [title] [description]"
	}

	assignedTo, err := strconv.ParseUint(args[0], 10, 32)
	if err != nil {
		return "❌ Invalid user ID"
	}

	task := &models.Task{
		Title:       args[1],
		Description: args[2],
		AssignedTo:  uint(assignedTo),
		Status:      string(models.Pending),
		Priority:    string(models.Medium),
		CreatedBy:   userID,
	}

	err = h.taskService.CreateDailyTask(task)
	if err != nil {
		return "❌ Failed to create daily task: " + err.Error()
	}

	return "✅ Daily task created successfully"
}

func (h *WhatsAppHandler) createMonthlyTask(userID uint, args []string) string {
	if len(args) < 3 {
		return "❌ Usage: /create_monthly_task [user_id] [title] [description]"
	}

	assignedTo, err := strconv.ParseUint(args[0], 10, 32)
	if err != nil {
		return "❌ Invalid user ID"
	}

	task := &models.Task{
		Title:       args[1],
		Description: args[2],
		AssignedTo:  uint(assignedTo),
		Status:      string(models.Pending),
		Priority:    string(models.Medium),
		CreatedBy:   userID,
	}

	err = h.taskService.CreateMonthlyTask(task)
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
