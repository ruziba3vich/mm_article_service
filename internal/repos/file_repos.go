package repos

import "context"

type MinIOStorage interface {
	CreateFile(ctx context.Context, fileName string, fileContent []byte) (string, string, error)
	DeleteFile(ctx context.Context, fileName string) error
	GetFileURL(ctx context.Context, fileName string) (string, error)
}
