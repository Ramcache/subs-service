package subscription

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

type ServiceImpl struct {
	repo Repository
}

func NewService(repo Repository) *ServiceImpl {
	return &ServiceImpl{repo: repo}
}

func (s *ServiceImpl) Create(ctx context.Context, req CreateRequest) (SubscriptionResponse, error) {
	sub, err := mapCreateToDomain(req)
	if err != nil {
		return SubscriptionResponse{}, err
	}

	created, err := s.repo.Create(ctx, sub)
	if err != nil {
		return SubscriptionResponse{}, err
	}

	return mapDomainToResponse(created), nil
}

func (s *ServiceImpl) Get(ctx context.Context, id string) (SubscriptionResponse, error) {
	if strings.TrimSpace(id) == "" {
		return SubscriptionResponse{}, ErrInvalidInput
	}
	got, err := s.repo.Get(ctx, id)
	if err != nil {
		return SubscriptionResponse{}, err
	}
	return mapDomainToResponse(got), nil
}

func (s *ServiceImpl) Update(ctx context.Context, id string, req UpdateRequest) (SubscriptionResponse, error) {
	if strings.TrimSpace(id) == "" {
		return SubscriptionResponse{}, ErrInvalidInput
	}

	// читаем текущую запись, применяем patch
	cur, err := s.repo.Get(ctx, id)
	if err != nil {
		return SubscriptionResponse{}, err
	}

	if req.ServiceName != nil {
		cur.ServiceName = strings.TrimSpace(*req.ServiceName)
	}
	if req.Price != nil {
		cur.Price = *req.Price
	}
	if req.UserID != nil {
		cur.UserID = strings.TrimSpace(*req.UserID)
	}
	if req.StartDate != nil {
		m, err := ParseMonthMMYYYY(*req.StartDate)
		if err != nil {
			return SubscriptionResponse{}, ErrInvalidInput
		}
		cur.StartMonth = m
	}
	if req.EndDate != nil {
		m, err := ParseMonthMMYYYY(*req.EndDate)
		if err != nil {
			return SubscriptionResponse{}, ErrInvalidInput
		}
		cur.EndMonth = &m
	}

	// финальная валидация
	if err := validateDomain(cur); err != nil {
		return SubscriptionResponse{}, err
	}

	updated, err := s.repo.Update(ctx, cur)
	if err != nil {
		return SubscriptionResponse{}, err
	}
	return mapDomainToResponse(updated), nil
}

func (s *ServiceImpl) Delete(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return ErrInvalidInput
	}
	return s.repo.Delete(ctx, id)
}

func (s *ServiceImpl) List(ctx context.Context, userID, serviceName *string, limit, offset int) (ListResponse, error) {
	items, total, err := s.repo.List(ctx, userID, serviceName, limit, offset)
	if err != nil {
		return ListResponse{}, err
	}

	resp := ListResponse{
		Items:  make([]SubscriptionResponse, 0, len(items)),
		Limit:  limit,
		Offset: offset,
		Total:  total,
	}
	for _, it := range items {
		resp.Items = append(resp.Items, mapDomainToResponse(it))
	}
	return resp, nil
}

func (s *ServiceImpl) Total(ctx context.Context, from, to string, userID, serviceName *string) (TotalResponse, error) {
	fm, err := ParseMonthMMYYYY(from)
	if err != nil {
		return TotalResponse{}, ErrInvalidInput
	}
	tm, err := ParseMonthMMYYYY(to)
	if err != nil {
		return TotalResponse{}, ErrInvalidInput
	}
	if tm.Compare(fm) < 0 {
		return TotalResponse{}, ErrInvalidInput
	}

	total, err := s.repo.Total(ctx, fm, tm, userID, serviceName)
	if err != nil {
		return TotalResponse{}, err
	}

	return TotalResponse{
		From:        from,
		To:          to,
		UserID:      userID,
		ServiceName: serviceName,
		Total:       total,
	}, nil
}

// --- mapping/validation helpers

func mapCreateToDomain(req CreateRequest) (Subscription, error) {
	if err := validateCreate(req); err != nil {
		return Subscription{}, ErrInvalidInput
	}
	if _, err := uuid.Parse(strings.TrimSpace(req.UserID)); err != nil {
		return Subscription{}, ErrInvalidInput
	}

	sm, _ := ParseMonthMMYYYY(req.StartDate)

	var em *Month
	if req.EndDate != nil {
		m, _ := ParseMonthMMYYYY(*req.EndDate)
		em = &m
	}

	sub := Subscription{
		ServiceName: strings.TrimSpace(req.ServiceName),
		Price:       req.Price,
		UserID:      strings.TrimSpace(req.UserID),
		StartMonth:  sm,
		EndMonth:    em,
	}
	if err := validateDomain(sub); err != nil {
		return Subscription{}, err
	}
	return sub, nil
}

func validateDomain(s Subscription) error {
	if strings.TrimSpace(s.ServiceName) == "" {
		return ErrInvalidInput
	}
	if s.Price < 0 {
		return ErrInvalidInput
	}
	if _, err := uuid.Parse(strings.TrimSpace(s.UserID)); err != nil {
		return ErrInvalidInput
	}
	if s.EndMonth != nil && s.EndMonth.Compare(s.StartMonth) < 0 {
		return ErrInvalidInput
	}
	return nil
}

func mapDomainToResponse(s Subscription) SubscriptionResponse {
	var end *string
	if s.EndMonth != nil {
		v := s.EndMonth.String()
		end = &v
	}
	return SubscriptionResponse{
		ID:          s.ID,
		ServiceName: s.ServiceName,
		Price:       s.Price,
		UserID:      s.UserID,
		StartDate:   s.StartMonth.String(),
		EndDate:     end,
	}
}

// чтобы не зависеть от stub файла
var _ Service = (*ServiceImpl)(nil)

// преобразуем “сырые” repo ошибки в сервисные
func wrapNotFound(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	return err
}
