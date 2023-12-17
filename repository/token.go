package repository

import (
	"log"

	"github.com/voice0726/oauth-playground/model"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"moul.io/zapgorm2"
)

type TokenRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewTokenRepository(dsn string, lg *zap.Logger) (*TokenRepository, error) {
	zg := zapgorm2.New(lg)
	zg.SetAsDefault()
	zg.LogLevel = gormlogger.Error
	zg.IgnoreRecordNotFoundError = true
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: zg})
	if err != nil {
		return nil, err
	}
	return &TokenRepository{db: db, logger: lg}, nil
}

func (r *TokenRepository) Create(token model.Token) (*model.Token, error) {
	log.Print(token)
	if err := r.db.Create(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}
