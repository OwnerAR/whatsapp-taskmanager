package main

import (
	"fmt"
	"log"
	"task_manager/internal/config"
	"task_manager/internal/database"
	"task_manager/internal/models"
	"task_manager/internal/redis"
	"task_manager/internal/repository"
	"task_manager/internal/services"
	"task_manager/pkg/whatsapp"

	"gorm.io/gorm"
)

func main() {
	fmt.Println("Initializing database...")

	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Initialize(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Force recreate all tables
	fmt.Println("Dropping existing tables...")
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

	// Create tables with proper schema
	fmt.Println("Creating tables...")
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

	// Create default super admin user
	fmt.Println("Creating default super admin user...")
	userRepo := repository.NewUserRepository(db)
	userService := services.NewUserService(userRepo)

	// Check if super admin already exists
	existingUser, err := userService.GetUserByUsername("admin")
	if err == nil && existingUser != nil {
		fmt.Println("Super admin user already exists")
		return
	}

	// Create super admin user
	superAdmin := &models.User{
		Username:       "admin",
		Email:          "egatryagung@gmail.com",
		PhoneNumber:    "6289502333331",
		Role:           string(models.SuperAdmin),
		WhatsAppNumber: "6289502333331",
		IsActive:       true,
	}

	err = userService.CreateUser(superAdmin, "admin123")
	if err != nil {
		log.Printf("Warning: Failed to create super admin user: %v", err)
	} else {
		fmt.Println("Super admin user created successfully")
		fmt.Println("Username: admin")
		fmt.Println("Password: admin123")
		fmt.Println("WhatsApp: 6281234567890")
	}

	// Create default financial settings
	fmt.Println("Creating default financial settings...")
	financialRepo := repository.NewFinancialRepository(db)

	// Tax rate setting
	taxSetting := &models.FinancialSettings{
		SettingName:     "tax_rate",
		PercentageValue: 10.0,
		IsPercentage:    true,
		IsActive:        true,
		CreatedBy:       1, // Super admin ID
	}
	financialRepo.CreateSettings(taxSetting)

	// Marketing rate setting
	marketingSetting := &models.FinancialSettings{
		SettingName:     "marketing_rate",
		PercentageValue: 5.0,
		IsPercentage:    true,
		IsActive:        true,
		CreatedBy:       1, // Super admin ID
	}
	financialRepo.CreateSettings(marketingSetting)

	// Rental rate setting
	rentalSetting := &models.FinancialSettings{
		SettingName:     "rental_rate",
		PercentageValue: 3.0,
		IsPercentage:    true,
		IsActive:        true,
		CreatedBy:       1, // Super admin ID
	}
	financialRepo.CreateSettings(rentalSetting)

	fmt.Println("Database initialization completed successfully!")
}
