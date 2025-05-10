package service

import (
	"context"
	"fmt"

	"github.com/ruziba3vich/mm_article_service/genprotos/genprotos/article_protos"
	"github.com/ruziba3vich/mm_article_service/genprotos/genprotos/user_protos"
	"github.com/ruziba3vich/mm_article_service/internal/models"
	"github.com/ruziba3vich/mm_article_service/internal/repos"
	logger "github.com/ruziba3vich/prodonik_lgger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	ArticleService struct {
		storage repos.ArticleRepo
		logger  *logger.Logger
		article_protos.UnimplementedArticleServiceServer
		filesStorage  repos.MinIOStorage
		userService   user_protos.UserServiceClient
		fileDbStorage repos.PictureRepo
	}
)

func NewArticleService(storage repos.ArticleRepo,
	logger *logger.Logger,
	filesStorage repos.MinIOStorage,
	userService user_protos.UserServiceClient,
	fileDbStorage repos.PictureRepo) *ArticleService {
	return &ArticleService{
		logger:        logger,
		filesStorage:  filesStorage,
		userService:   userService,
		storage:       storage,
		fileDbStorage: fileDbStorage,
	}
}

func (a *ArticleService) CreateArticle(ctx context.Context, req *article_protos.CreateArticleRequest) (*article_protos.ArticleEntity, error) {
	article, err := a.storage.CreateArticle(ctx, req)
	if err != nil {
		a.logger.Error("failed to create article", map[string]any{"user_id": req.UserId, "error": err.Error()})
		return nil, err
	}
	files := make([]*article_protos.FileEntity, len(req.Files))

	for i := range req.Files {
		fileName, url, err := a.filesStorage.CreateFile(ctx, req.Files[i].Name, req.Files[i].Content)
		if err != nil {
			a.logger.Error("failed to create file in MinIO", map[string]any{"file_name": req.Files[i].Name, "error": err.Error()})
			return nil, err
		}
		files[i] = &article_protos.FileEntity{
			FileName: fileName,
			Url:      url,
		}
		if err := a.fileDbStorage.CreatePicture(ctx, &models.Picture{FileName: fileName, ArticleID: article.Id}); err != nil {
			a.logger.Error("failed to store picture in database", map[string]any{"file_name": fileName, "article_id": article.Id, "error": err.Error()})
			return nil, err
		}
	}
	article.Files = files
	if err := a.fillArticleEntity(ctx, article, req.UserId); err != nil {
		a.logger.Error("failed to fill article entity", map[string]any{"user_id": req.UserId, "article_id": article.Id, "error": err.Error()})
		return nil, err
	}

	return article, nil
}

func (a *ArticleService) DeleteArticle(ctx context.Context, req *article_protos.DeleteArticleRequest) (*article_protos.DeleteArticleResponse, error) {
	article, err := a.storage.GetArticleByID(ctx, &article_protos.GetArticleByIDRequest{ArticleId: req.ArticleId})
	if err != nil {
		a.logger.Error("failed to fetch article for deletion", map[string]any{"article_id": req.ArticleId, "error": err.Error()})
		return nil, fmt.Errorf("could not fetch article: %s", err.Error())
	}
	go func() {
		for i := range article.Article.Files {
			if err := a.filesStorage.DeleteFile(ctx, article.Article.Files[i].FileName); err != nil {
				a.logger.Error("failed to delete file from MinIO", map[string]any{"file_name": article.Article.Files[i].FileName, "error": err.Error()})
			}
			if err := a.fileDbStorage.DeletePicture(ctx, article.Article.Files[i].FileName, article.Article.Id); err != nil {
				a.logger.Error("failed to delete picture from database", map[string]any{"file_name": article.Article.Files[i].FileName, "article_id": article.Article.Id, "error": err.Error()})
			}
		}
	}()
	resp, err := a.storage.DeleteArticle(ctx, req)
	if err != nil {
		a.logger.Error("failed to delete article", map[string]any{"article_id": req.ArticleId, "error": err.Error()})
		return nil, err
	}
	return resp, nil
}

func (a *ArticleService) GetArticleByID(ctx context.Context, req *article_protos.GetArticleByIDRequest) (*article_protos.GetArticleByIDResponse, error) {
	article, err := a.storage.GetArticleByID(ctx, req)
	if err != nil {
		a.logger.Error("failed to fetch article by ID", map[string]any{"article_id": req.ArticleId, "error": err.Error()})
		return nil, err
	}

	files, err := a.fileDbStorage.GetPicturesByArticle(ctx, article.Article.Id)
	if err != nil {
		a.logger.Error("failed to fetch pictures for article", map[string]any{"article_id": article.Article.Id, "error": err.Error()})
		return nil, err
	}
	for i := range files {
		fileUrl, err := a.filesStorage.GetFileURL(ctx, files[i].FileName)
		if err != nil {
			a.logger.Error("failed to get file URL from MinIO", map[string]any{"file_name": files[i].FileName, "article_id": article.Article.Id, "error": err.Error()})
			return nil, err
		}
		article.Article.Files = append(article.Article.Files, &article_protos.FileEntity{
			FileName: files[i].FileName,
			Url:      fileUrl,
		})
	}

	if err := a.fillArticleEntity(ctx, article.Article, article.Article.UserId); err != nil {
		a.logger.Error("failed to fill article entity", map[string]any{"user_id": article.Article.UserId, "article_id": article.Article.Id, "error": err.Error()})
		return nil, err
	}

	return article, nil
}

func (a *ArticleService) GetArticles(ctx context.Context, req *article_protos.GetArticlesRequest) (*article_protos.GetArticlesResponse, error) {
	resp, err := a.storage.GetArticles(ctx, req)
	if err != nil {
		a.logger.Error("failed to fetch articles", map[string]any{"page": req.Pagination.Page, "page_size": req.Pagination.PageSize, "error": err.Error()})
		return nil, err
	}
	for i := range resp.Pagination.Articles {
		if err := a.fillArticleEntity(ctx, resp.Pagination.Articles[i], resp.Pagination.Articles[i].UserId); err != nil {
			a.logger.Error("failed to fill article entity", map[string]any{"user_id": resp.Pagination.Articles[i].UserId, "article_id": resp.Pagination.Articles[i].Id, "error": err.Error()})
			return nil, err
		}
		files, err := a.fileDbStorage.GetPicturesByArticle(ctx, resp.Pagination.Articles[i].Id)
		if err != nil {
			a.logger.Error("failed to fetch pictures for article", map[string]any{"article_id": resp.Pagination.Articles[i].Id, "error": err.Error()})
			return nil, err
		}
		for j := range files {
			fileUrl, err := a.filesStorage.GetFileURL(ctx, files[j].FileName)
			if err != nil {
				a.logger.Error("failed to get file URL from MinIO", map[string]any{"file_name": files[j].FileName, "article_id": resp.Pagination.Articles[i].Id, "error": err.Error()})
				return nil, err
			}
			resp.Pagination.Articles[i].Files = append(resp.Pagination.Articles[i].Files, &article_protos.FileEntity{
				FileName: files[j].FileName,
				Url:      fileUrl,
			})
		}
	}

	return resp, nil
}

func (a *ArticleService) GetArticlesByUser(ctx context.Context, req *article_protos.GetArticlesByUserRequest) (*article_protos.GetArticlesByUserResponse, error) {
	resp, err := a.storage.GetArticlesByUser(ctx, req)
	if err != nil {
		a.logger.Error("failed to fetch articles by user", map[string]any{"user_id": req.UserId, "page": req.Pagination.Page, "page_size": req.Pagination.PageSize, "error": err.Error()})
		return nil, err
	}
	for i := range resp.Pagination.Articles {
		if err := a.fillArticleEntity(ctx, resp.Pagination.Articles[i], resp.Pagination.Articles[i].UserId); err != nil {
			a.logger.Error("failed to fill article entity", map[string]any{"user_id": resp.Pagination.Articles[i].UserId, "article_id": resp.Pagination.Articles[i].Id, "error": err.Error()})
			return nil, err
		}
		files, err := a.fileDbStorage.GetPicturesByArticle(ctx, resp.Pagination.Articles[i].Id)
		if err != nil {
			a.logger.Error("failed to fetch pictures for article", map[string]any{"article_id": resp.Pagination.Articles[i].Id, "error": err.Error()})
			return nil, err
		}
		for j := range files {
			fileUrl, err := a.filesStorage.GetFileURL(ctx, files[j].FileName)
			if err != nil {
				a.logger.Error("failed to get file URL from MinIO", map[string]any{"file_name": files[j].FileName, "article_id": resp.Pagination.Articles[i].Id, "error": err.Error()})
				return nil, err
			}
			resp.Pagination.Articles[i].Files = append(resp.Pagination.Articles[i].Files, &article_protos.FileEntity{
				FileName: files[j].FileName,
				Url:      fileUrl,
			})
		}
	}

	return resp, nil
}

func (a *ArticleService) LikeArticle(ctx context.Context, req *article_protos.LikeArticleRequest) (*article_protos.LikeArticleResponse, error) {
	if liked, err := a.storage.HasUserLikedArticle(ctx, req.UserId, req.ArticleId); err != nil {
		a.logger.Error("failed to check if user liked article", map[string]any{"user_id": req.UserId, "article_id": req.ArticleId, "error": err.Error()})
		return nil, err
	} else if liked {
		err := status.Error(codes.AlreadyExists, "you have already liked the post")
		a.logger.Error("user already liked the post", map[string]any{"user_id": req.UserId, "article_id": req.ArticleId, "error": err.Error()})
		return nil, err
	} else {
		resp, err := a.storage.LikeArticle(ctx, req)
		if err != nil {
			a.logger.Error("failed to like article", map[string]any{"user_id": req.UserId, "article_id": req.ArticleId, "error": err.Error()})
			return nil, err
		}
		return resp, nil
	}
}

func (a *ArticleService) RewriteArticle(ctx context.Context, req *article_protos.RewriteArticleRequest) (*article_protos.ArticleEntity, error) {
	article, err := a.storage.RewriteArticle(ctx, req)
	if err != nil {
		a.logger.Error("failed to rewrite article", map[string]any{"user_id": req.UserId, "original_article_id": req.OriginalArticleId, "error": err.Error()})
		return nil, err
	}
	files := make([]*article_protos.FileEntity, len(req.Files))
	for i := range req.Files {
		fileName, url, err := a.filesStorage.CreateFile(ctx, req.Files[i].Name, req.Files[i].Content)
		if err != nil {
			a.logger.Error("failed to create file in MinIO", map[string]any{"file_name": req.Files[i].Name, "error": err.Error()})
			return nil, err
		}
		files[i] = &article_protos.FileEntity{FileName: fileName, Url: url}
		if err := a.fileDbStorage.CreatePicture(ctx, &models.Picture{
			FileName:  fileName,
			ArticleID: article.Id,
		}); err != nil {
			a.logger.Error("failed to store picture in database", map[string]any{"file_name": fileName, "article_id": article.Id, "error": err.Error()})
			return nil, err
		}
	}
	return article, nil
}

func (a *ArticleService) UnlikeArticle(ctx context.Context, req *article_protos.UnlikeArticleRequest) (*article_protos.UnlikeArticleResponse, error) {
	if liked, err := a.storage.HasUserLikedArticle(ctx, req.UserId, req.ArticleId); err != nil {
		a.logger.Error("failed to check if user liked article", map[string]any{"user_id": req.UserId, "article_id": req.ArticleId, "error": err.Error()})
		return nil, err
	} else if !liked {
		err := status.Error(codes.FailedPrecondition, "you have not liked the post")
		a.logger.Error("user has not liked the post", map[string]any{"user_id": req.UserId, "article_id": req.ArticleId, "error": err.Error()})
		return nil, err
	} else {
		resp, err := a.storage.UnlikeArticle(ctx, req)
		if err != nil {
			a.logger.Error("failed to unlike article", map[string]any{"user_id": req.UserId, "article_id": req.ArticleId, "error": err.Error()})
			return nil, err
		}
		return resp, nil
	}
}

func (a *ArticleService) UpdateArticle(ctx context.Context, req *article_protos.UpdateArticleRequest) (*article_protos.ArticleEntity, error) {
	article, err := a.storage.UpdateArticle(ctx, req)
	if err != nil {
		a.logger.Error("failed to update article", map[string]any{"article_id": req.ArticleId, "error": err.Error()})
		return nil, err
	}
	return article, nil
}

func (a *ArticleService) fillArticleEntity(ctx context.Context, article *article_protos.ArticleEntity, userID string) error {
	userData, err := a.userService.GetUserData(ctx, &user_protos.GetUserDataRequest{UserId: userID})
	if err != nil {
		a.logger.Error("failed to fetch user data", map[string]any{"user_id": userID, "error": err.Error()})
		return err
	}

	article.UserFullName = userData.FullName
	article.UserUsername = userData.Username
	article.UserProfilePic = userData.ProfilePicUrl
	return nil
}
