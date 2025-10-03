package migrations

import (
	"log"
	"task_manager/internal/models"
	"task_manager/internal/repository"
	"task_manager/internal/services"

	"gorm.io/gorm"
)

// RunMigrations runs all database migrations and creates default data
func RunMigrations(db *gorm.DB) error {
	log.Println("Running database migrations...")

	// Force recreate all tables to ensure proper schema
	log.Println("Dropping existing tables...")
	err := db.Migrator().DropTable(
		&models.User{},
		&models.Task{},
		&models.TaskProgress{},
		&models.DailyTask{},
		&models.MonthlyTask{},
		&models.Order{},
		&models.OrderItem{},
		&models.Reminder{},
		&models.FinancialSettings{},
		&models.CalculationHistory{},
		&models.ReportQuery{},
	)
	if err != nil {
		log.Printf("Warning: Error dropping tables: %v", err)
	}

	// Create tables with proper schema
	log.Println("Creating tables...")
	err = db.AutoMigrate(
		&models.User{},
		&models.Task{},
		&models.TaskProgress{},
		&models.DailyTask{},
		&models.MonthlyTask{},
		&models.Order{},
		&models.OrderItem{},
		&models.Reminder{},
		&models.FinancialSettings{},
		&models.CalculationHistory{},
		&models.ReportQuery{},
	)
	if err != nil {
		return err
	}

	// Create default data
	err = createDefaultData(db)
	if err != nil {
		log.Printf("Warning: Failed to create default data: %v", err)
	}

	log.Println("Database migrations completed successfully!")
	return nil
}

// createDefaultData creates default users and settings
func createDefaultData(db *gorm.DB) error {
	log.Println("Creating default data...")

	// Initialize repositories and services
	userRepo := repository.NewUserRepository(db)
	userService := services.NewUserService(userRepo)
	financialRepo := repository.NewFinancialRepository(db)

	// Check if super admin already exists
	existingUser, err := userService.GetUserByUsername("admin")
	if err == nil && existingUser != nil {
		log.Println("Super admin user already exists")
		return nil
	}

	// Create super admin user
	log.Println("Creating super admin user...")
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
		log.Println("Super admin user created successfully")
		log.Println("Username: admin")
		log.Println("Password: admin123")
		log.Println("WhatsApp: 6289502333331")
	}

	// Create default financial settings
	log.Println("Creating default financial settings...")

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

	log.Println("Default data created successfully!")
	return nil
}
