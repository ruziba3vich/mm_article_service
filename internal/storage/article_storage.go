package storage

import (
	"context"
	"math/rand"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/ruziba3vich/mm_article_service/genprotos/genprotos/article_protos"
	"github.com/ruziba3vich/mm_article_service/internal/models"
	"github.com/ruziba3vich/mm_article_service/internal/repos"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// articleRepository implements ArticleRepo
type articleRepository struct {
	db *gorm.DB
}

// NewArticleRepository creates a new articleRepository
func NewArticleRepository(db *gorm.DB) repos.ArticleRepo {
	return &articleRepository{db: db}
}

// CreateArticle stores a new article
func (r *articleRepository) CreateArticle(ctx context.Context, in *article_protos.CreateArticleRequest) (*article_protos.ArticleEntity, error) {
	files := make([]string, len(in.Files))
	for i, f := range in.Files {
		files[i] = f.Name // TODO: Service layer should provide URLs
	}
	article := models.Article{
		ID:      generateULID(),
		UserID:  in.UserId,
		Title:   in.Title,
		Content: in.Content,
	}
	if err := r.db.WithContext(ctx).Create(&article).Error; err != nil {
		return nil, err
	}

	return article.ToArticleEntity(), nil
}

// UpdateArticle updates an article
func (r *articleRepository) UpdateArticle(ctx context.Context, in *article_protos.UpdateArticleRequest) (*article_protos.ArticleEntity, error) {
	updates := map[string]any{
		"title":      in.Title,
		"content":    in.Content,
		"updated_at": time.Now(),
	}
	result := r.db.WithContext(ctx).Model(&models.Article{}).Where("id = ?", in.ArticleId).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	var article models.Article
	if err := r.db.WithContext(ctx).Where("id = ?", in.ArticleId).First(&article).Error; err != nil {
		return nil, err
	}
	return article.ToArticleEntity(), nil
}

// RewriteArticle stores a new article with original_article_id
func (r *articleRepository) RewriteArticle(ctx context.Context, in *article_protos.RewriteArticleRequest) (*article_protos.ArticleEntity, error) {
	article := models.Article{
		UserID:            in.UserId,
		OriginalArticleID: in.OriginalArticleId,
		Title:             in.Title,
		Content:           in.Content,
	}
	if err := r.db.WithContext(ctx).Create(&article).Error; err != nil {
		return nil, err
	}
	return article.ToArticleEntity(), nil
}

// DeleteArticle deletes an article
func (r *articleRepository) DeleteArticle(ctx context.Context, in *article_protos.DeleteArticleRequest) (*article_protos.DeleteArticleResponse, error) {
	result := r.db.WithContext(ctx).Where("id = ?", in.ArticleId).Delete(&models.Article{})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &article_protos.DeleteArticleResponse{Success: true}, nil
}

// LikeArticle adds a like
func (r *articleRepository) LikeArticle(ctx context.Context, in *article_protos.LikeArticleRequest) (*article_protos.LikeArticleResponse, error) {
	like := models.ArticleLike{
		UserID:    in.UserId,
		ArticleID: in.ArticleId,
	}
	if err := r.db.WithContext(ctx).Create(&like).Error; err != nil {
		return nil, err
	}
	return &article_protos.LikeArticleResponse{Success: true}, nil
}

// UnlikeArticle removes a like
func (r *articleRepository) UnlikeArticle(ctx context.Context, in *article_protos.UnlikeArticleRequest) (*article_protos.UnlikeArticleResponse, error) {
	result := r.db.WithContext(ctx).Where("user_id = ? AND article_id = ?", in.UserId, in.ArticleId).Delete(&models.ArticleLike{})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &article_protos.UnlikeArticleResponse{Success: true}, nil
}

// GetArticlesByUser fetches articles by user
func (r *articleRepository) GetArticlesByUser(ctx context.Context, in *article_protos.GetArticlesByUserRequest) (*article_protos.GetArticlesByUserResponse, error) {
	var articles []models.Article
	var totalCount int64
	offset := (in.Pagination.Page - 1) * in.Pagination.PageSize

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "SET TRANSACTION ISOLATION LEVEL REPEATABLE READ"}).
			Model(&models.Article{}).Where("user_id = ?", in.UserId).Count(&totalCount).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", in.UserId).
			Offset(int(offset)).Limit(int(in.Pagination.PageSize)).Order("created_at DESC").Find(&articles).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	protoArticles := make([]*article_protos.ArticleEntity, len(articles))
	for i, a := range articles {
		protoArticles[i] = &article_protos.ArticleEntity{
			Id:                a.ID,
			UserId:            a.UserID,
			OriginalArticleId: a.OriginalArticleID,
			Title:             a.Title,
			Content:           a.Content,
			CreatedAt:         timestamppb.New(a.CreatedAt),
			LikeCount:         int32(a.LikesCount),
		}
	}
	return &article_protos.GetArticlesByUserResponse{
		Pagination: &article_protos.PaginationResponse{
			Articles:   protoArticles,
			TotalCount: int32(totalCount),
			Page:       in.Pagination.Page,
			PageSize:   in.Pagination.PageSize,
		},
	}, nil
}

// GetArticles fetches all articles
func (r *articleRepository) GetArticles(ctx context.Context, in *article_protos.GetArticlesRequest) (*article_protos.GetArticlesResponse, error) {
	var articles []models.Article
	var totalCount int64
	offset := (in.Pagination.Page - 1) * in.Pagination.PageSize

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "SET TRANSACTION ISOLATION LEVEL REPEATABLE READ"}).
			Model(&models.Article{}).Count(&totalCount).Error; err != nil {
			return err
		}
		if err := tx.Offset(int(offset)).Limit(int(in.Pagination.PageSize)).Order("created_at DESC").Find(&articles).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	protoArticles := make([]*article_protos.ArticleEntity, len(articles))
	for i, a := range articles {
		protoArticles[i] = &article_protos.ArticleEntity{
			Id:                a.ID,
			UserId:            a.UserID,
			OriginalArticleId: a.OriginalArticleID,
			Title:             a.Title,
			Content:           a.Content,
			CreatedAt:         timestamppb.New(a.CreatedAt),
			LikeCount:         int32(a.LikesCount),
		}
	}
	return &article_protos.GetArticlesResponse{
		Pagination: &article_protos.PaginationResponse{
			Articles:   protoArticles,
			TotalCount: int32(totalCount),
			Page:       in.Pagination.Page,
			PageSize:   in.Pagination.PageSize,
		},
	}, nil
}

// GetArticleByID fetches a single article
func (r *articleRepository) GetArticleByID(ctx context.Context, in *article_protos.GetArticleByIDRequest) (*article_protos.ArticleEntity, error) {
	var article models.Article
	if err := r.db.WithContext(ctx).Where("id = ?", in.ArticleId).First(&article).Error; err != nil {
		return nil, err
	}
	return article.ToArticleEntity(), nil
}

func generateULID() string {
	t := time.Now()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
	return ulid.MustNew(ulid.Timestamp(t), entropy).String()
}
