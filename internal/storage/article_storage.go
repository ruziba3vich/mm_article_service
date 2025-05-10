package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/k0kubun/pp"
	"github.com/ruziba3vich/mm_article_service/genprotos/genprotos/article_protos"
	"github.com/ruziba3vich/mm_article_service/internal/models"
	"github.com/ruziba3vich/mm_article_service/internal/repos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
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
	if in.UserId == "" || in.Title == "" || in.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id, title, and content are required")
	}

	article := models.Article{
		ID:      generateULID(),
		UserID:  in.UserId,
		Title:   in.Title,
		Content: in.Content,
	}
	if err := r.db.WithContext(ctx).Create(&article).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create article: %v", err)
	}

	return article.ToArticleEntity(), nil
}

// UpdateArticle updates an article
func (r *articleRepository) UpdateArticle(ctx context.Context, in *article_protos.UpdateArticleRequest) (*article_protos.ArticleEntity, error) {
	if in.ArticleId == "" || in.Title == "" || in.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "article_id, title, and content are required")
	}

	updates := map[string]any{
		"title":      in.Title,
		"content":    in.Content,
		"updated_at": time.Now(),
	}
	result := r.db.WithContext(ctx).Model(&models.Article{}).Where("id = ?", in.ArticleId).Updates(updates)
	if result.Error != nil {
		return nil, status.Errorf(codes.Internal, "failed to update article: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "article not found")
	}

	var article models.Article
	if err := r.db.WithContext(ctx).Where("id = ?", in.ArticleId).First(&article).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "article not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to fetch updated article: %v", err)
	}
	return article.ToArticleEntity(), nil
}

// RewriteArticle stores a new article with original_article_id
func (r *articleRepository) RewriteArticle(ctx context.Context, in *article_protos.RewriteArticleRequest) (*article_protos.ArticleEntity, error) {
	if in.UserId == "" || in.OriginalArticleId == "" || in.Title == "" || in.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id, original_article_id, title, and content are required")
	}

	// Verify original article exists
	var original models.Article
	if err := r.db.WithContext(ctx).Where("id = ?", in.OriginalArticleId).First(&original).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "original article not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to verify original article: %v", err)
	}

	article := models.Article{
		ID:                generateULID(),
		UserID:            in.UserId,
		OriginalArticleID: in.OriginalArticleId,
		Title:             in.Title,
		Content:           in.Content,
	}
	if err := r.db.WithContext(ctx).Create(&article).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create rewritten article: %v", err)
	}
	return article.ToArticleEntity(), nil
}

// DeleteArticle deletes an article
func (r *articleRepository) DeleteArticle(ctx context.Context, in *article_protos.DeleteArticleRequest) (*article_protos.DeleteArticleResponse, error) {
	if in.ArticleId == "" {
		return nil, status.Error(codes.InvalidArgument, "article_id is required")
	}

	result := r.db.WithContext(ctx).Where("id = ?", in.ArticleId).Delete(&models.Article{})
	if result.Error != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete article: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "article not found")
	}
	return &article_protos.DeleteArticleResponse{Success: true}, nil
}

// LikeArticle adds a like
func (r *articleRepository) LikeArticle(ctx context.Context, in *article_protos.LikeArticleRequest) (*article_protos.LikeArticleResponse, error) {
	if in.UserId == "" || in.ArticleId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and article_id are required")
	}

	// Check if the article exists
	var article models.Article
	if err := r.db.WithContext(ctx).Where("id = ?", in.ArticleId).First(&article).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "article not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to verify article: %v", err)
	}

	// Check if already liked
	hasLiked, err := r.HasUserLikedArticle(ctx, in.UserId, in.ArticleId)
	if err != nil {
		return nil, err
	}
	if hasLiked {
		return nil, status.Error(codes.AlreadyExists, "user has already liked this article")
	}

	like := models.ArticleLike{
		UserID:    in.UserId,
		ArticleID: in.ArticleId,
	}
	if err := r.db.WithContext(ctx).Create(&like).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create like: %v", err)
	}
	return &article_protos.LikeArticleResponse{Success: true}, nil
}

// UnlikeArticle removes a like
func (r *articleRepository) UnlikeArticle(ctx context.Context, in *article_protos.UnlikeArticleRequest) (*article_protos.UnlikeArticleResponse, error) {
	if in.UserId == "" || in.ArticleId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and article_id are required")
	}

	result := r.db.WithContext(ctx).Where("user_id = ? AND article_id = ?", in.UserId, in.ArticleId).Delete(&models.ArticleLike{})
	if result.Error != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove like: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "like not found")
	}
	return &article_protos.UnlikeArticleResponse{Success: true}, nil
}

// GetArticlesByUser fetches articles by user
func (r *articleRepository) GetArticlesByUser(ctx context.Context, in *article_protos.GetArticlesByUserRequest) (*article_protos.GetArticlesByUserResponse, error) {
	if in.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if in.Pagination.Page <= 0 || in.Pagination.PageSize <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid pagination parameters")
	}

	var articles []models.Article
	var totalCount int64
	offset := (in.Pagination.Page - 1) * in.Pagination.PageSize

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET TRANSACTION ISOLATION LEVEL REPEATABLE READ").Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Article{}).Where("user_id = ?", in.UserId).Count(&totalCount).Error; err != nil {
			return err
		}

		if err := tx.Where("user_id = ?", in.UserId).
			Offset(int(offset)).Limit(int(in.Pagination.PageSize)).Order("created_at DESC").Find(&articles).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch articles: %v", err)
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
	if in.Pagination.Page <= 0 || in.Pagination.PageSize <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid pagination parameters")
	}

	var articles []models.Article
	var totalCount int64
	offset := (in.Pagination.Page - 1) * in.Pagination.PageSize

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET TRANSACTION ISOLATION LEVEL REPEATABLE READ").Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Article{}).Count(&totalCount).Error; err != nil {
			return err
		}

		if err := tx.Offset(int(offset)).Limit(int(in.Pagination.PageSize)).Order("created_at DESC").Find(&articles).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch articles: %v", err)
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
func (r *articleRepository) GetArticleByID(ctx context.Context, in *article_protos.GetArticleByIDRequest) (*article_protos.GetArticleByIDResponse, error) {
	if in.ArticleId == "" {
		return nil, status.Error(codes.InvalidArgument, "article_id is required")
	}

	var article models.Article
	if err := r.db.WithContext(ctx).Where("id = ?", in.ArticleId).First(&article).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "article not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to fetch article: %v", err)
	}
	pp.Println(article)
	return &article_protos.GetArticleByIDResponse{
		Article: article.ToArticleEntity(),
	}, nil
}

// HasUserLikedArticle checks if a user has liked an article
func (r *articleRepository) HasUserLikedArticle(ctx context.Context, userID, articleID string) (bool, error) {
	if userID == "" || articleID == "" {
		return false, status.Error(codes.InvalidArgument, "user_id and article_id are required")
	}

	var count int64
	if err := r.db.WithContext(ctx).Model(&models.ArticleLike{}).
		Where("user_id = ? AND article_id = ?", userID, articleID).
		Count(&count).Error; err != nil {
		return false, status.Errorf(codes.Internal, "failed to check like status: %v", err)
	}
	return count > 0, nil
}

func generateULID() string {
	now := time.Now()
	timeComponent := fmt.Sprintf("%04d%02d%02d%02d%02d%02d%09d",
		now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(), now.Second(),
		now.Nanosecond())
	if len(timeComponent) > 32 {
		timeComponent = timeComponent[:32]
	}
	for len(timeComponent) < 32 {
		timeComponent += "0"
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		timeComponent[0:8],
		timeComponent[8:12],
		timeComponent[12:16],
		timeComponent[16:20],
		timeComponent[20:32])
}
