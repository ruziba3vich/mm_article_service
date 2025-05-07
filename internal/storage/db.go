package storage

import (
	"github.com/ruziba3vich/mm_article_service/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// NewGORM initializes a GORM database connection with migrations
func NewGORM(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&models.Article{}, &models.ArticleLike{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
