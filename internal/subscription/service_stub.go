package subscription

import (
	"context"
)

type StubService struct{}

func NewStubService() *StubService { return &StubService{} }

func (s *StubService) Create(ctx context.Context, req CreateRequest) (SubscriptionResponse, error) {
	_ = ctx
	resp := SubscriptionResponse{
		ID:          "00000000-0000-0000-0000-000000000000",
		ServiceName: req.ServiceName,
		Price:       req.Price,
		UserID:      req.UserID,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
	}
	return resp, nil
}

func (s *StubService) Get(ctx context.Context, id string) (SubscriptionResponse, error) {
	_ = ctx
	return SubscriptionResponse{
		ID:          id,
		ServiceName: "stub",
		Price:       0,
		UserID:      "00000000-0000-0000-0000-000000000000",
		StartDate:   "01-2000",
	}, nil
}

func (s *StubService) Update(ctx context.Context, id string, req UpdateRequest) (SubscriptionResponse, error) {
	_ = ctx
	_ = req
	return SubscriptionResponse{
		ID:          id,
		ServiceName: "stub",
		Price:       0,
		UserID:      "00000000-0000-0000-0000-000000000000",
		StartDate:   "01-2000",
	}, nil
}

func (s *StubService) Delete(ctx context.Context, id string) error {
	_ = ctx
	_ = id
	return nil
}

func (s *StubService) List(ctx context.Context, userID, serviceName *string, limit, offset int) (ListResponse, error) {
	_ = ctx
	_ = userID
	_ = serviceName
	return ListResponse{
		Items:  []SubscriptionResponse{},
		Limit:  limit,
		Offset: offset,
		Total:  0,
	}, nil
}

func (s *StubService) Total(ctx context.Context, from, to string, userID, serviceName *string) (TotalResponse, error) {
	_ = ctx
	return TotalResponse{
		From:        from,
		To:          to,
		UserID:      userID,
		ServiceName: serviceName,
		Total:       0,
	}, nil
}
