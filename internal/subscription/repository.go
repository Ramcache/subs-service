package subscription

import "context"

type Repository interface {
	Create(ctx context.Context, s Subscription) (Subscription, error)
	Get(ctx context.Context, id string) (Subscription, error)
	Update(ctx context.Context, s Subscription) (Subscription, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, userID, serviceName *string, limit, offset int) (items []Subscription, total int64, err error)
	Total(ctx context.Context, from Month, to Month, userID, serviceName *string) (int64, error)
}
