package repos

import (
	"context"

	"github.com/ruziba3vich/mm_article_service/genprotos/genprotos/article_protos"
)

type ArticleRepo interface {
	CreateArticle(context.Context, *article_protos.CreateArticleRequest) (*article_protos.ArticleEntity, error)
	UpdateArticle(context.Context, *article_protos.UpdateArticleRequest) (*article_protos.ArticleEntity, error)
	RewriteArticle(context.Context, *article_protos.RewriteArticleRequest) (*article_protos.ArticleEntity, error)
	DeleteArticle(context.Context, *article_protos.DeleteArticleRequest) (*article_protos.DeleteArticleResponse, error)
	LikeArticle(context.Context, *article_protos.LikeArticleRequest) (*article_protos.LikeArticleResponse, error)
	UnlikeArticle(context.Context, *article_protos.UnlikeArticleRequest) (*article_protos.UnlikeArticleResponse, error)
	GetArticlesByUser(context.Context, *article_protos.GetArticlesByUserRequest) (*article_protos.GetArticlesByUserResponse, error)
	GetArticles(context.Context, *article_protos.GetArticlesRequest) (*article_protos.GetArticlesResponse, error)
	GetArticleByID(context.Context, *article_protos.GetArticleByIDRequest) (*article_protos.GetArticleByIDResponse, error)
	HasUserLikedArticle(context.Context, string, string) (bool, error)
}
