package storage

import (
	"context"

	"github.com/ruziba3vich/mm_article_service/internal/models"
	"gorm.io/gorm"
)

type (
	FileDbStorage struct {
		db *gorm.DB
	}
)

func (r *FileDbStorage) CreatePicture(ctx context.Context, picture *models.Picture) error {
	return r.db.WithContext(ctx).Create(picture).Error
}

func NewFileDbStorage(db *gorm.DB) *FileDbStorage {
	return &FileDbStorage{
		db: db,
	}
}

// GetPicture retrieves a picture by file_name and article_id
func (r *FileDbStorage) GetPicture(ctx context.Context, fileName, articleID string) (*models.Picture, error) {
	var picture models.Picture
	if err := r.db.WithContext(ctx).Where("file_name = ? AND article_id = ?", fileName, articleID).First(&picture).Error; err != nil {
		return nil, err
	}
	return &picture, nil
}

// GetPicturesByArticle fetches all pictures for an article
func (r *FileDbStorage) GetPicturesByArticle(ctx context.Context, articleID string) ([]*models.Picture, error) {
	var pictures []*models.Picture
	if err := r.db.WithContext(ctx).Where("article_id = ?", articleID).Find(&pictures).Error; err != nil {
		return nil, err
	}
	return pictures, nil
}

// DeletePicture removes a picture by file_name and article_id
func (r *FileDbStorage) DeletePicture(ctx context.Context, fileName, articleID string) error {
	result := r.db.WithContext(ctx).Where("file_name = ? AND article_id = ?", fileName, articleID).Delete(&models.Picture{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
