package storage

import (
	"github.com/ruziba3vich/mm_article_service/internal/models"
	"github.com/ruziba3vich/mm_article_service/pkg/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// NewGORM initializes a GORM database connection with migrations
func NewGORM(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.PsqlCfg.Dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&models.Article{}, &models.ArticleLike{}, &models.Picture{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
