// Super Admin/Admin command implementations
func (h *WhatsAppHandler) updateUser(args []string) string {
	if len(args) < 2 {
		return "❌ Usage: /update_user [user_id] [field]=[value] ..."
	}
	userID, err := strconv.ParseUint(args[0], 10, 32)
	if err != nil {
		return "❌ Invalid user ID"
	}
	user, err := h.userService.GetUserByID(uint(userID))
	if err != nil {
		return "❌ User not found"
	}
	for _, field := range args[1:] {
		kv := strings.SplitN(field, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "username":
			user.Username = kv[1]
		case "email":
			user.Email = kv[1]
		case "phone":
			user.PhoneNumber = kv[1]
			user.WhatsAppNumber = kv[1]
		case "role":
			user.Role = kv[1]
		}
	}
	err = h.userService.UpdateUser(user)
	if err != nil {
		return "❌ Failed to update user: " + err.Error()
	}
	return "✅ User updated successfully"
}

func (h *WhatsAppHandler) deleteUser(args []string) string {
	if len(args) < 1 {
		return "❌ Usage: /delete_user [user_id]"
	}
	userID, err := strconv.ParseUint(args[0], 10, 32)
	if err != nil {
		return "❌ Invalid user ID"
	}
	err = h.userService.DeleteUser(uint(userID))
	if err != nil {
		return "❌ Failed to delete user: " + err.Error()
	}
	return "✅ User deleted successfully"
}

func (h *WhatsAppHandler) setRole(args []string) string {
	if len(args) < 2 {
		return "❌ Usage: /set_role [user_id] [role]"
	}
	userID, err := strconv.ParseUint(args[0], 10, 32)
	if err != nil {
		return "❌ Invalid user ID"
	}
	user, err := h.userService.GetUserByID(uint(userID))
	if err != nil {
		return "❌ User not found"
	}
	user.Role = args[1]
	err = h.userService.UpdateUser(user)
	if err != nil {
		return "❌ Failed to set role: " + err.Error()
	}
	return "✅ User role updated"
}

func (h *WhatsAppHandler) systemConfig(args []string) string {
	// Placeholder for system config logic
	return "⚙️ System configuration updated (not implemented)"
}

func (h *WhatsAppHandler) updateOrder(userID uint, args []string) string {
	if len(args) < 2 {
		return "❌ Usage: /update_order [order_id] [field]=[value] ..."
	}
	orderID, err := strconv.ParseUint(args[0], 10, 32)
	if err != nil {
		return "❌ Invalid order ID"
	}
	order, err := h.orderService.GetOrderByID(uint(orderID))
	if err != nil {
		return "❌ Order not found"
	}
	for _, field := range args[1:] {
		kv := strings.SplitN(field, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "customer_name":
			order.CustomerName = kv[1]
		case "total_amount":
			amt, err := strconv.ParseFloat(kv[1], 64)
			if err == nil {
				order.TotalAmount = amt
			}
		case "status":
			order.Status = kv[1]
		}
	}
	err = h.orderService.UpdateOrder(order)
	if err != nil {
		return "❌ Failed to update order: " + err.Error()
	}
	return "✅ Order updated successfully"
}

func (h *WhatsAppHandler) deleteOrder(args []string) string {
	if len(args) < 1 {
		return "❌ Usage: /delete_order [order_id]"
	}
	orderID, err := strconv.ParseUint(args[0], 10, 32)
	if err != nil {
		return "❌ Invalid order ID"
	}
	err = h.orderService.DeleteOrder(uint(orderID))
	if err != nil {
		return "❌ Failed to delete order: " + err.Error()
	}
	return "✅ Order deleted successfully"
}
package handlers
import (
	"encoding/json"
	"net/http"
	"github.com/gin-gonic/gin"
	"task_manager/internal/models"
	"task_manager/internal/services"
	"task_manager/internal/redis"
	"task_manager/internal/config"
	"strings"
	"fmt"
	"time"
	"strconv"
	"crypto/hmac"
	"crypto/sha256"
)
// Removed extra closing parenthesis

type WhatsAppHandler struct {
	whatsappService services.WhatsAppService
	userService     services.UserService
	taskService     services.TaskService
	orderService    services.OrderService
	reminderService services.ReminderService
	config          *config.Config
}

func NewWhatsAppHandler(
	whatsappService services.WhatsAppService,
	userService services.UserService,
	taskService services.TaskService,
	orderService services.OrderService,
	reminderService services.ReminderService,
	cfg *config.Config) *WhatsAppHandler {
	return &WhatsAppHandler{
		whatsappService: whatsappService,
		userService:     userService,
		taskService:     taskService,
		orderService:    orderService,
		reminderService: reminderService,
		config:          cfg,
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
	// HMAC signature verification
	signature := c.GetHeader("X-Hub-Signature-256")
	secret := h.config.WhatsappWebhookSecret
	body, err := c.GetRawData()
	fmt.Printf("[DEBUG] Signature: %v\n", signature)
	fmt.Printf("[DEBUG] Secret: %v\n", secret)
	fmt.Printf("[DEBUG] Raw body: %s\n", string(body))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}
	if !verifyWebhookSignature(body, signature, secret) {
		fmt.Println("[DEBUG] Signature verification failed")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	// Parse payload (support all event/message types)
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
		return
	}

	// Logging for debug
	fmt.Printf("Received webhook: %v\n", payload)

	// Handle event/message types
	if event, ok := payload["event"].(string); ok {
		switch event {
		case "message.ack":
			ack := payload["payload"].(map[string]interface{})
			fmt.Printf("Message %v: chat_id=%v, ids=%v, desc=%v\n",
				ack["receipt_type"], ack["chat_id"], ack["ids"], ack["receipt_type_description"])
		case "group.participants":
			group := payload["payload"].(map[string]interface{})
			fmt.Printf("Group %v event: chat_id=%v, users=%v\n",
				group["type"], group["chat_id"], group["jids"])
		default:
			fmt.Printf("Unhandled event: %v\n", event)
		}
	} else if action, ok := payload["action"].(string); ok {
		switch action {
		case "message_revoked":
			fmt.Printf("Message revoked: %v\n", payload["revoked_message_id"])
		case "message_edited":
			fmt.Printf("Message edited: %v\n", payload["edited_text"])
		default:
			fmt.Printf("Unhandled action: %v\n", action)
		}
	} else if msg, ok := payload["message"].(map[string]interface{}); ok {
		// Text, reply, reaction, media, etc
		text, _ := msg["text"].(string)
		senderID, _ := payload["sender_id"].(string)
	// pushname, _ := payload["pushname"].(string) // not used
		fmt.Printf("New message: %v from %v\n", text, senderID)

		// Only process if text starts with '/'
		if strings.HasPrefix(text, "/") {
			// Find user by WhatsApp number
			user, err := h.userService.GetUserByWhatsAppNumber(senderID)
			var reply string
			if err != nil || user == nil {
				reply = "❌ User not registered. Please contact admin."
			} else {
				reply = h.processCommand(user, text)
			}
			// Send WhatsApp reply
			errSend := h.whatsappService.SendMessage(senderID, reply)
			if errSend != nil {
				fmt.Printf("Failed to send WhatsApp reply: %v\n", errSend)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "OK"})
}

// verifyWebhookSignature verifies HMAC SHA256 signature from header
func verifyWebhookSignature(payload []byte, signature, secret string) bool {
	if signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := fmt.Sprintf("%x", mac.Sum(nil))
	received := strings.Replace(signature, "sha256=", "", 1)
	return hmac.Equal([]byte(expected), []byte(received))
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
	case "/daily_progress_reminder":
		return h.sendDailyProgressReminder(user)
	case "/monthly_progress_reminder":
		return h.sendMonthlyProgressReminder(user)
	default:
		// Check if user is admin or super admin for admin commands
		if user.Role == string(models.Admin) || user.Role == string(models.SuperAdmin) {
			return h.processAdminCommand(user, command, args)
		}
		return "❌ Unknown command. Type /help for available commands."
	}
}
// Kirim notifikasi progres harian ke user
func (h *WhatsAppHandler) sendDailyProgressReminder(user *models.User) string {
	// Ambil progres harian dari Redis atau TaskService
	progress := 0
	// TODO: ambil progres harian sebenarnya dari TaskService/Redis
	err := h.reminderService.SendDailyProgressReminder(user.WhatsAppNumber, progress)
	if err != nil {
		return "❌ Failed to send daily progress reminder: " + err.Error()
	}
	return "✅ Daily progress reminder sent"
}

// Kirim notifikasi progres bulanan ke user
func (h *WhatsAppHandler) sendMonthlyProgressReminder(user *models.User) string {
	// Ambil progres bulanan dari Redis atau TaskService
	progress := 0
	// TODO: ambil progres bulanan sebenarnya dari TaskService/Redis
	err := h.reminderService.SendMonthlyProgressReminder(user.WhatsAppNumber, progress)
	if err != nil {
		return "❌ Failed to send monthly progress reminder: " + err.Error()
	}
	return "✅ Monthly progress reminder sent"
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
	 case "/update_user":
		return h.updateUser(args)
	 case "/delete_user":
		return h.deleteUser(args)
	 case "/set_role":
		return h.setRole(args)
	 case "/system_config":
		return h.systemConfig(args)
	 case "/update_order":
		return h.updateOrder(user.ID, args)
	 case "/delete_order":
		return h.deleteOrder(args)
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
	settings := &models.FinancialSettings{
		SettingName:    "tax_rate",
		PercentageValue: percentage,
		IsPercentage:   true,
		IsActive:       true,
		CreatedBy:      userID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	// Gunakan repository langsung
	if repo, ok := h.orderService.(interface{ CreateFinancialSettings(*models.FinancialSettings) error }); ok {
		err = repo.CreateFinancialSettings(settings)
		if err != nil {
			return "❌ Failed to set tax rate: " + err.Error()
		}
		return fmt.Sprintf("✅ Tax rate set to %.2f%%", percentage)
	}
	return "❌ Financial repository not available"
}

func (h *WhatsAppHandler) setMarketingRate(userID uint, args []string) string {
	if len(args) < 1 {
		return "❌ Usage: /set_marketing_rate [percentage]"
	}
	percentage, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return "❌ Invalid percentage"
	}
	settings := &models.FinancialSettings{
		SettingName:    "marketing_rate",
		PercentageValue: percentage,
		IsPercentage:   true,
		IsActive:       true,
		CreatedBy:      userID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if repo, ok := h.orderService.(interface{ CreateFinancialSettings(*models.FinancialSettings) error }); ok {
		err = repo.CreateFinancialSettings(settings)
		if err != nil {
			return "❌ Failed to set marketing rate: " + err.Error()
		}
		return fmt.Sprintf("✅ Marketing rate set to %.2f%%", percentage)
	}
	return "❌ Financial repository not available"
}

func (h *WhatsAppHandler) setRentalRate(userID uint, args []string) string {
	if len(args) < 1 {
		return "❌ Usage: /set_rental_rate [percentage]"
	}
	percentage, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return "❌ Invalid percentage"
	}
	settings := &models.FinancialSettings{
		SettingName:    "rental_rate",
		PercentageValue: percentage,
		IsPercentage:   true,
		IsActive:       true,
		CreatedBy:      userID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if repo, ok := h.orderService.(interface{ CreateFinancialSettings(*models.FinancialSettings) error }); ok {
		err = repo.CreateFinancialSettings(settings)
		if err != nil {
			return "❌ Failed to set rental rate: " + err.Error()
		}
		return fmt.Sprintf("✅ Rental rate set to %.2f%%", percentage)
	}
	return "❌ Financial repository not available"
}

func (h *WhatsAppHandler) generateReport() string {
	// Ambil semua order dan hitung total, pajak, marketing, rental, net profit
	orders, err := h.orderService.GetAllOrders()
	if err != nil {
		return "❌ Failed to get orders: " + err.Error()
	}
	totalAmount := 0.0
	totalTax := 0.0
	totalMarketing := 0.0
	totalRental := 0.0
	totalNetProfit := 0.0
	for _, order := range orders {
		totalAmount += order.TotalAmount
		totalTax += order.TaxAmount
		totalMarketing += order.MarketingCost
		totalRental += order.RentalCost
		totalNetProfit += order.NetProfit
	}
	response := "📊 **Financial Report:**\n\n"
	response += fmt.Sprintf("Total Orders: %d\n", len(orders))
	response += fmt.Sprintf("Total Amount: $%.2f\n", totalAmount)
	response += fmt.Sprintf("Total Tax: $%.2f\n", totalTax)
	response += fmt.Sprintf("Total Marketing: $%.2f\n", totalMarketing)
	response += fmt.Sprintf("Total Rental: $%.2f\n", totalRental)
	response += fmt.Sprintf("Net Profit: $%.2f\n", totalNetProfit)
	return response
}

func (h *WhatsAppHandler) generateDailyReport() string {
	// Ambil order hari ini
	today := time.Now().Format("2006-01-02")
	start, _ := time.Parse("2006-01-02", today)
	end := start.Add(24 * time.Hour)
	orders, err := h.orderService.GetOrdersByDateRange(start, end)
	if err != nil {
		return "❌ Failed to get daily orders: " + err.Error()
	}
	totalAmount := 0.0
	totalNetProfit := 0.0
	for _, order := range orders {
		totalAmount += order.TotalAmount
		totalNetProfit += order.NetProfit
	}
	response := "📊 **Daily Report:**\n\n"
	response += fmt.Sprintf("Total Orders: %d\n", len(orders))
	response += fmt.Sprintf("Total Amount: $%.2f\n", totalAmount)
	response += fmt.Sprintf("Net Profit: $%.2f\n", totalNetProfit)
	return response
}

func (h *WhatsAppHandler) generateMonthlyReport() string {
	// Ambil order bulan ini
	month := time.Now().Format("2006-01")
	start, _ := time.Parse("2006-01-02", month+"-01")
	end := start.AddDate(0, 1, 0)
	orders, err := h.orderService.GetOrdersByDateRange(start, end)
	if err != nil {
		return "❌ Failed to get monthly orders: " + err.Error()
	}
	totalAmount := 0.0
	totalNetProfit := 0.0
	for _, order := range orders {
		totalAmount += order.TotalAmount
		totalNetProfit += order.NetProfit
	}
	response := "📊 **Monthly Report:**\n\n"
	response += fmt.Sprintf("Total Orders: %d\n", len(orders))
	response += fmt.Sprintf("Total Amount: $%.2f\n", totalAmount)
	response += fmt.Sprintf("Net Profit: $%.2f\n", totalNetProfit)
	return response
}
