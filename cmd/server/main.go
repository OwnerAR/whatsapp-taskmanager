package main

import (
	"log"
	"task_manager/internal/config"
	"task_manager/internal/database"
	"task_manager/internal/handlers"
	"task_manager/internal/models"
	"task_manager/internal/redis"
	"task_manager/internal/repository"
	"task_manager/internal/services"
	"task_manager/pkg/whatsapp"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Initialize(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Force recreate tables to ensure proper schema
	log.Println("Ensuring database schema is up to date...")
	err = db.Migrator().DropTable(
		&models.User{},
		&models.Task{},
		&models.TaskProgress{},
		&models.DailyTask{},
		&models.MonthlyTask{},
		&models.Order{},
		&models.Reminder{},
		&models.FinancialSettings{},
		&models.CalculationHistory{},
		&models.ReportQuery{},
	)
	if err != nil {
		log.Printf("Warning: Error dropping tables: %v", err)
	}

	// Recreate tables with proper schema
	err = db.AutoMigrate(
		&models.User{},
		&models.Task{},
		&models.TaskProgress{},
		&models.DailyTask{},
		&models.MonthlyTask{},
		&models.Order{},
		&models.Reminder{},
		&models.FinancialSettings{},
		&models.CalculationHistory{},
		&models.ReportQuery{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Initialize Redis
	redisClient, err := redis.Initialize(cfg.RedisURL)
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}

	// Initialize WhatsApp client
	whatsappClient := whatsapp.NewClient(cfg.WhatsAppAPIURL, cfg.WhatsAppUsername, cfg.WhatsAppPassword, cfg.WhatsAppPath)

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	reminderRepo := repository.NewReminderRepository(db)
	financialRepo := repository.NewFinancialRepository(db)

	// Initialize services
	userService := services.NewUserService(userRepo)
	taskService := services.NewTaskService(taskRepo, redisClient)
	orderService := services.NewOrderService(orderRepo, financialRepo)
	whatsappService := services.NewWhatsAppService(whatsappClient, redisClient)
	reminderService := services.NewReminderService(reminderRepo, whatsappService)

	// Initialize handlers
	whatsappHandler := handlers.NewWhatsAppHandler(whatsappService, userService, taskService, orderService, reminderService)
	apiHandler := handlers.NewAPIHandler(userService, taskService, orderService)

	// Setup routes
	router := gin.Default()
	
	// WhatsApp webhook
	router.POST("/api/whatsapp/webhook", whatsappHandler.HandleWebhook)
	router.POST("/api/whatsapp/send-message", whatsappHandler.SendMessage)
	
	// API endpoints
	api := router.Group("/api")
	{
		api.POST("/whatsapp/interactive-session", whatsappHandler.StartInteractiveSession)
		api.PUT("/whatsapp/session/:session_id", whatsappHandler.UpdateSession)
		api.DELETE("/whatsapp/session/:session_id", whatsappHandler.EndSession)
		
		// Cache endpoints
		api.GET("/cache/session/:session_id", apiHandler.GetSession)
		api.POST("/cache/session", apiHandler.CreateSession)
		api.PUT("/cache/session/:session_id", apiHandler.UpdateSession)
		api.DELETE("/cache/session/:session_id", apiHandler.DeleteSession)
		
		api.GET("/cache/temp-data/:key", apiHandler.GetTempData)
		api.POST("/cache/temp-data", apiHandler.StoreTempData)
		api.DELETE("/cache/temp-data/:key", apiHandler.DeleteTempData)
	}

	// Start server
	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
