package models

import "time"

type Article struct {
	ID                string    `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserID            string    `gorm:"type:uuid;not null"`
	OriginalArticleID string    `gorm:"type:uuid"`
	Title             string    `gorm:"not null"`
	Content           string    `gorm:"not null"`
	CreatedAt         time.Time `gorm:"autoCreateTime"`
	FileURLs          []string  `gorm:"type:text[]"`
}

type ArticleLike struct {
	UserID    string    `gorm:"type:uuid;not null;primaryKey"`
	ArticleID string    `gorm:"type:uuid;not null;primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}
