package models

import (
	"time"

	"github.com/ruziba3vich/mm_article_service/genprotos/genprotos/article_protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Article struct {
	ID                string    `gorm:"primaryKey;type:uuid;"`
	UserID            string    `gorm:"type:uuid;not null"`
	OriginalArticleID string    `gorm:"type:uuid"`
	Title             string    `gorm:"not null"`
	Content           string    `gorm:"not null"`
	CreatedAt         time.Time `gorm:"autoCreateTime"`
	LikesCount        int       `gorm:"not null;default:0"`
}

type ArticleLike struct {
	UserID    string    `gorm:"type:uuid;not null;primaryKey"`
	ArticleID string    `gorm:"type:uuid;not null;primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (a *Article) ToArticleEntity() *article_protos.ArticleEntity {
	return &article_protos.ArticleEntity{
		Id:        a.ID,
		UserId:    a.UserID,
		Title:     a.Title,
		Content:   a.Content,
		CreatedAt: timestamppb.New(a.CreatedAt),
		LikeCount: int32(a.LikesCount),
	}
}
