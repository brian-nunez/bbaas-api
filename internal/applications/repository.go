package applications

import "context"

type Repository interface {
	Create(ctx context.Context, app Application, tokenDigest string) error
	GetByID(ctx context.Context, id string) (Application, bool, error)
	GetByTokenDigest(ctx context.Context, tokenDigest string) (Application, bool, error)
}
