package repos

import (
	"context"

	"github.com/ruziba3vich/mm_article_service/internal/models"
)

type PictureRepo interface {
	CreatePicture(ctx context.Context, picture *models.Picture) error
	GetPicture(ctx context.Context, fileName, articleID string) (*models.Picture, error)
	GetPicturesByArticle(ctx context.Context, articleID string) ([]*models.Picture, error)
	DeletePicture(ctx context.Context, fileName, articleID string) error
}
