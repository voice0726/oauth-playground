package repository

import (
	"github.com/voice0726/oauth-playground/model"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"moul.io/zapgorm2"
)

type AuthRequestRepository struct {
	db *gorm.DB
	lg *zap.Logger
}

func NewAuthRequestRepository(dsn string, lg *zap.Logger) (*AuthRequestRepository, error) {
	zg := zapgorm2.New(lg)
	zg.SetAsDefault()
	zg.LogLevel = gormlogger.Error
	zg.IgnoreRecordNotFoundError = true
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: zg})

	if err != nil {
		return nil, err
	}

	return &AuthRequestRepository{db: db, lg: lg}, nil
}

func (r *AuthRequestRepository) CreateRequest(req model.AuthRequest) (*model.AuthRequest, error) {
	if err := r.db.Save(&req).Error; err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *AuthRequestRepository) FindRequestByID(ID string) (*model.AuthRequest, error) {
	var result model.AuthRequest
	if err := r.db.Model(&model.AuthRequest{}).Where("id = ?", ID).First(&result).Error; err != nil {
		return nil, err
	}
	return &result, nil
}
