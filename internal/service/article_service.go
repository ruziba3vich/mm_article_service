package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/ruziba3vich/mm_article_service/genprotos/genprotos/article_protos"
	"github.com/ruziba3vich/mm_article_service/genprotos/genprotos/user_protos"
	"github.com/ruziba3vich/mm_article_service/internal/repos"
	logger "github.com/ruziba3vich/prodonik_lgger"
)

type (
	ArticleService struct {
		storage repos.ArticleRepo
		logger  *logger.Logger
		article_protos.UnimplementedArticleServiceServer
		filesStorage repos.MinIOStorage
		userService  user_protos.UserServiceClient
	}
)

func NewArticleService(storage repos.ArticleRepo,
	logger *logger.Logger,
	filesStorage repos.MinIOStorage,
	userService user_protos.UserServiceClient) *ArticleService {
	return &ArticleService{
		logger:       logger,
		filesStorage: filesStorage,
		userService:  userService,
		storage:      storage,
	}
}

func (a *ArticleService) CreateArticle(ctx context.Context, req *article_protos.CreateArticleRequest) (*article_protos.ArticleEntity, error) {

	files := make([]*article_protos.FileEntity, len(req.Files))

	for i := range req.Files {
		fileName, url, err := a.filesStorage.CreateFile(ctx, req.Files[i].Name, req.Files[i].Content)
		if err != nil {
			return nil, err
		}
		files[i] = &article_protos.FileEntity{
			FileName: fileName,
			Url:      url,
		}
	}
	article, err := a.storage.CreateArticle(ctx, req)
	if err != nil {
		return nil, err
	}
	article.Files = files
	if err := a.fillArticleEntity(ctx, article, req.UserId); err != nil {
		a.logger.Println(err)
		return nil, err
	}

	return article, nil
}

func (a *ArticleService) DeleteArticle(ctx context.Context, req *article_protos.DeleteArticleRequest) (*article_protos.DeleteArticleResponse, error) {
	article, err := a.storage.GetArticleByID(ctx, &article_protos.GetArticleByIDRequest{ArticleId: req.ArticleId})
	if err != nil {
		return nil, fmt.Errorf("could not fetch article: %s", err.Error())
	}
	go func() {
		for i := range article.Files {
			a.filesStorage.DeleteFile(ctx, article.Files[i].FileName)
		}
	}()
	resp, err := a.storage.DeleteArticle(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (a *ArticleService) GetArticleByID(ctx context.Context, req *article_protos.GetArticleByIDRequest) (*article_protos.ArticleEntity, error) {
	article, err := a.storage.GetArticleByID(ctx, req)
	if err != nil {
		return nil, err
	}

	if err := a.fillArticleEntity(ctx, article, article.UserId); err != nil {
		return nil, err
	}

	return article, nil
}

func (a *ArticleService) GetArticles(ctx context.Context, req *article_protos.GetArticlesRequest) (*article_protos.GetArticlesResponse, error) {
	resp, err := a.storage.GetArticles(ctx, req)
	if err != nil {
		return nil, err
	}
	for i := range resp.Pagination.Articles {
		if err := a.fillArticleEntity(ctx, resp.Pagination.Articles[i], resp.Pagination.Articles[i].UserId); err != nil {
			return nil, err
		}
	}

	return resp, nil
}

func (a *ArticleService) GetArticlesByUser(ctx context.Context, req *article_protos.GetArticlesByUserRequest) (*article_protos.GetArticlesByUserResponse, error) {
	resp, err := a.storage.GetArticlesByUser(ctx, req)
	if err != nil {
		return nil, err
	}
	for i := range resp.Pagination.Articles {
		if err := a.fillArticleEntity(ctx, resp.Pagination.Articles[i], resp.Pagination.Articles[i].UserId); err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// func (a *ArticleService) LikeArticle(context.Context, *article_protos.LikeArticleRequest) (*article_protos.LikeArticleResponse, error)
func (a *ArticleService) RewriteArticle(ctx context.Context, req *article_protos.RewriteArticleRequest) (*article_protos.ArticleEntity, error) {
	files := make([]*article_protos.FileEntity, len(req.Files))
	for i := range req.Files {
		fileName, url, err := a.filesStorage.CreateFile(ctx, req.Files[i].Name, req.Files[i].Content)
		if err != nil {
			return nil, err
		}
		files[i] = &article_protos.FileEntity{FileName: fileName, Url: url}
	}
	article, err := a.storage.RewriteArticle(ctx, req)
	if err != nil {
		return nil, err
	}
	return article, nil
}

func (a *ArticleService) UnlikeArticle(ctx context.Context, req *article_protos.UnlikeArticleRequest) (*article_protos.ArticleEntity, error) {
	if liked, err := a.storage.HasUserLikedArticle(ctx, req.UserId, req.ArticleId); err != nil {
		return nil, err

	} else if !liked {
		return nil, errors.New("you have not liked the post")
	} else {
		resp, err := a.UnlikeArticle(ctx, req)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}
}

// func (a *ArticleService) UpdateArticle(context.Context, *article_protos.UpdateArticleRequest) (*article_protos.ArticleEntity, error)

func (a *ArticleService) fillArticleEntity(ctx context.Context, article *article_protos.ArticleEntity, userID string) error {
	userData, err := a.userService.GetUserData(ctx, &user_protos.GetUserDataRequest{UserId: userID})
	if err != nil {
		return err
	}

	article.UserFullName = userData.FullName
	article.UserUsername = userData.Username
	article.UserProfilePic = userData.ProfilePicUrl
	return nil
}
