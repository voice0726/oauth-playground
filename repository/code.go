package repository

import (
	"github.com/voice0726/oauth-playground/model"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"moul.io/zapgorm2"
)

type CodeRepository struct {
	db *gorm.DB
	lg *zap.Logger
}

func NewCodeRepository(dsn string, lg *zap.Logger) (*CodeRepository, error) {
	zg := zapgorm2.New(lg)
	zg.SetAsDefault()
	zg.LogLevel = gormlogger.Error
	zg.IgnoreRecordNotFoundError = true
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: zg})
	if err != nil {
		return nil, err
	}
	return &CodeRepository{db: db, lg: lg}, nil
}

func (r *CodeRepository) FindByID(ID string) (*model.AuthCode, error) {
	var result *model.AuthCode
	if err := r.db.Model(&model.AuthCode{}).Where("id = ?", ID).First(&result).Error; err != nil {
		return nil, err
	}

	return result, nil
}

func (r *CodeRepository) FindByCode(code string) (*model.AuthCode, error) {
	var result *model.AuthCode
	if err := r.db.Model(&model.AuthCode{}).Where("code = ?", code).First(&result).Error; err != nil {
		return nil, err
	}

	return result, nil

}

func (r *CodeRepository) Create(code model.AuthCode) (*model.AuthCode, error) {
	if err := r.db.Model(&model.AuthCode{}).Save(&code).Error; err != nil {
		return nil, err
	}
	return &code, nil
}
