package repository

import (
	"task_manager/internal/models"

	"gorm.io/gorm"
)

type FinancialRepository interface {
	CreateSettings(settings *models.FinancialSettings) error
	GetSettings(settingName string) (*models.FinancialSettings, error)
	UpdateSettings(settings *models.FinancialSettings) error
	CreateCalculationHistory(history *models.CalculationHistory) error
	GetCalculationHistory(orderID uint) ([]models.CalculationHistory, error)
	CreateReportQuery(query *models.ReportQuery) error
	GetReportQuery(id uint) (*models.ReportQuery, error)
}

type financialRepository struct {
	db *gorm.DB
}

func NewFinancialRepository(db *gorm.DB) FinancialRepository {
	return &financialRepository{db: db}
}

func (r *financialRepository) CreateSettings(settings *models.FinancialSettings) error {
	return r.db.Create(settings).Error
}

func (r *financialRepository) GetSettings(settingName string) (*models.FinancialSettings, error) {
	var settings models.FinancialSettings
	err := r.db.Where("setting_name = ? AND is_active = ?", settingName, true).First(&settings).Error
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

func (r *financialRepository) UpdateSettings(settings *models.FinancialSettings) error {
	return r.db.Save(settings).Error
}

func (r *financialRepository) CreateCalculationHistory(history *models.CalculationHistory) error {
	return r.db.Create(history).Error
}

func (r *financialRepository) GetCalculationHistory(orderID uint) ([]models.CalculationHistory, error) {
	var history []models.CalculationHistory
	err := r.db.Where("order_id = ?", orderID).Find(&history).Error
	return history, err
}

func (r *financialRepository) CreateReportQuery(query *models.ReportQuery) error {
	return r.db.Create(query).Error
}

func (r *financialRepository) GetReportQuery(id uint) (*models.ReportQuery, error) {
	var query models.ReportQuery
	err := r.db.First(&query, id).Error
	if err != nil {
		return nil, err
	}
	return &query, nil
}
