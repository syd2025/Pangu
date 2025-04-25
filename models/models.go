package models

import (
	"context"
	"time"

	"example.com/myapp/utils/helper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Models struct {
	Users  UserModel
	Tokens TokenModel
}

func NewModels(cfg *DatabaseConfig, helper *helper.Helper) (*Models, error) {
	db, err := gorm.Open(postgres.Open(cfg.Dsn()), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 迁移数据库
	db.AutoMigrate(&User{}, &UserBrief{}, &UserProfile{})

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// 线程池配置
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxIdleTime(cfg.MaxIdleDuration())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = sqlDB.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return &Models{
		Users:  UserModel{DB: db, helper: helper},
		Tokens: TokenModel{DB: db},
	}, nil
}
