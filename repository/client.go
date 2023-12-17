package repository

import (
	"github.com/voice0726/oauth-playground/model"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"moul.io/zapgorm2"
)

var ErrClientNotFound error

type ClientRepository struct {
	db *gorm.DB
	lg *zap.Logger
}

func NewClientRepository(dsn string, lg *zap.Logger) (*ClientRepository, error) {
	zg := zapgorm2.New(lg)
	zg.SetAsDefault()
	zg.LogLevel = gormlogger.Error
	zg.IgnoreRecordNotFoundError = true
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: zg})
	if err != nil {
		return nil, err
	}
	return &ClientRepository{db: db, lg: lg}, nil
}

func (r *ClientRepository) FindClientByID(ID string) (*model.Client, error) {
	var result *model.Client
	if err := r.db.Model(&model.Client{}).Where("id = ?", ID).First(&result).Error; err != nil {
		return nil, err
	}

	return result, nil
}

func (r *ClientRepository) FindClientByName(name string) (*model.Client, error) {
	var result *model.Client
	if err := r.db.Model(&model.Client{}).Where("name = ?", name).First(&result).Error; err != nil {
		return nil, err
	}

	return result, nil
}
