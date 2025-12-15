package subscription

import (
	"context"
	"errors"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
)

type Service interface {
	Create(ctx context.Context, req CreateRequest) (SubscriptionResponse, error)
	Get(ctx context.Context, id string) (SubscriptionResponse, error)
	Update(ctx context.Context, id string, req UpdateRequest) (SubscriptionResponse, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, userID, serviceName *string, limit, offset int) (ListResponse, error)
	Total(ctx context.Context, from, to string, userID, serviceName *string) (TotalResponse, error)
}
