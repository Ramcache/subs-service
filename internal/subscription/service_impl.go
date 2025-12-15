package subscription

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	plog "subs-service/internal/platform/logger"
)

type ServiceImpl struct {
	repo Repository
}

func NewService(repo Repository) *ServiceImpl {
	return &ServiceImpl{repo: repo}
}

type ValidationError struct {
	Msg string
}

func (e ValidationError) Error() string { return e.Msg }

func (s *ServiceImpl) Create(ctx context.Context, req CreateRequest) (SubscriptionResponse, error) {
	log := plog.FromContext(ctx).With(
		zap.String("op", "subscription.create"),
		zap.String("user_id", strings.TrimSpace(req.UserID)),
		zap.String("service_name", strings.TrimSpace(req.ServiceName)),
		zap.Int64("price", req.Price),
		zap.String("start_date", strings.TrimSpace(req.StartDate)),
		zap.Bool("has_end_date", req.EndDate != nil),
	)

	log.Debug("create started")

	if strings.TrimSpace(req.ServiceName) == "" {
		log.Warn("validation failed", zap.String("reason", "service_name is required"))
		return SubscriptionResponse{}, ValidationError{"service_name is required"}
	}
	if req.Price < 0 {
		log.Warn("validation failed", zap.String("reason", "price must be >= 0"))
		return SubscriptionResponse{}, ValidationError{"price must be >= 0"}
	}
	if _, err := uuid.Parse(req.UserID); err != nil {
		log.Warn("validation failed", zap.String("reason", "invalid user_id"), zap.Error(err))
		return SubscriptionResponse{}, ValidationError{"invalid user_id"}
	}

	start, err := ParseMonthMMYYYY(req.StartDate)
	if err != nil {
		log.Warn("validation failed", zap.String("reason", "invalid start_date"), zap.Error(err))
		return SubscriptionResponse{}, ValidationError{"invalid start_date"}
	}

	var end *Month
	if req.EndDate != nil {
		m, err := ParseMonthMMYYYY(*req.EndDate)
		if err != nil {
			log.Warn("validation failed", zap.String("reason", "invalid end_date"), zap.Error(err))
			return SubscriptionResponse{}, ValidationError{"invalid end_date"}
		}
		if m.Compare(start) < 0 {
			log.Warn("validation failed", zap.String("reason", "end_date must be >= start_date"),
				zap.String("end_date", strings.TrimSpace(*req.EndDate)),
			)
			return SubscriptionResponse{}, ValidationError{"end_date must be >= start_date"}
		}
		end = &m
	}

	sub := Subscription{
		ServiceName: strings.TrimSpace(req.ServiceName),
		Price:       req.Price,
		UserID:      strings.TrimSpace(req.UserID),
		StartMonth:  start,
		EndMonth:    end,
	}

	log.Debug("calling repository create")
	created, err := s.repo.Create(ctx, sub)
	if err != nil {
		log.Error("repository create failed", zap.Error(err))
		return SubscriptionResponse{}, err
	}

	resp := mapDomainToResponse(created)
	log.Info("create succeeded", zap.String("subscription_id", resp.ID))
	return resp, nil
}

func (s *ServiceImpl) Get(ctx context.Context, id string) (SubscriptionResponse, error) {
	id = strings.TrimSpace(id)

	log := plog.FromContext(ctx).With(
		zap.String("op", "subscription.get"),
		zap.String("subscription_id", id),
	)

	log.Debug("get started")

	if id == "" {
		log.Warn("validation failed", zap.String("reason", "id is required"))
		return SubscriptionResponse{}, ErrInvalidInput
	}

	log.Debug("calling repository get")
	got, err := s.repo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			log.Info("subscription not found")
		} else {
			log.Error("repository get failed", zap.Error(err))
		}
		return SubscriptionResponse{}, err
	}

	resp := mapDomainToResponse(got)
	log.Debug("get succeeded")
	return resp, nil
}

func (s *ServiceImpl) Update(ctx context.Context, id string, req UpdateRequest) (SubscriptionResponse, error) {
	id = strings.TrimSpace(id)

	log := plog.FromContext(ctx).With(
		zap.String("op", "subscription.update"),
		zap.String("subscription_id", id),
		zap.Bool("patch_service_name", req.ServiceName != nil),
		zap.Bool("patch_price", req.Price != nil),
		zap.Bool("patch_user_id", req.UserID != nil),
		zap.Bool("patch_start_date", req.StartDate != nil),
		zap.Bool("patch_end_date", req.EndDate != nil),
	)

	log.Debug("update started")

	if id == "" {
		log.Warn("validation failed", zap.String("reason", "id is required"))
		return SubscriptionResponse{}, ErrInvalidInput
	}

	log.Debug("calling repository get (load current)")
	cur, err := s.repo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			log.Info("subscription not found")
		} else {
			log.Error("repository get failed", zap.Error(err))
		}
		return SubscriptionResponse{}, err
	}

	if req.ServiceName != nil {
		cur.ServiceName = strings.TrimSpace(*req.ServiceName)
		log.Debug("patched field", zap.String("field", "service_name"))
	}
	if req.Price != nil {
		cur.Price = *req.Price
		log.Debug("patched field", zap.String("field", "price"), zap.Int64("new_price", cur.Price))
	}
	if req.UserID != nil {
		cur.UserID = strings.TrimSpace(*req.UserID)
		log.Debug("patched field", zap.String("field", "user_id"))
	}
	if req.StartDate != nil {
		m, err := ParseMonthMMYYYY(*req.StartDate)
		if err != nil {
			log.Warn("validation failed", zap.String("reason", "invalid start_date"), zap.Error(err))
			return SubscriptionResponse{}, ErrInvalidInput
		}
		cur.StartMonth = m
		log.Debug("patched field", zap.String("field", "start_date"), zap.String("new_start_date", m.String()))
	}
	if req.EndDate != nil {
		m, err := ParseMonthMMYYYY(*req.EndDate)
		if err != nil {
			log.Warn("validation failed", zap.String("reason", "invalid end_date"), zap.Error(err))
			return SubscriptionResponse{}, ErrInvalidInput
		}
		cur.EndMonth = &m
		log.Debug("patched field", zap.String("field", "end_date"), zap.String("new_end_date", m.String()))
	}

	log.Debug("validating updated domain")
	if err := validateDomain(cur); err != nil {
		log.Warn("validation failed", zap.String("reason", "domain validation failed"), zap.Error(err))
		return SubscriptionResponse{}, err
	}

	log.Debug("calling repository update")
	updated, err := s.repo.Update(ctx, cur)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			log.Info("subscription not found on update")
		} else {
			log.Error("repository update failed", zap.Error(err))
		}
		return SubscriptionResponse{}, err
	}

	resp := mapDomainToResponse(updated)
	log.Info("update succeeded", zap.String("subscription_id", resp.ID))
	return resp, nil
}

func (s *ServiceImpl) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)

	log := plog.FromContext(ctx).With(
		zap.String("op", "subscription.delete"),
		zap.String("subscription_id", id),
	)

	log.Debug("delete started")

	if id == "" {
		log.Warn("validation failed", zap.String("reason", "id is required"))
		return ErrInvalidInput
	}

	log.Debug("calling repository delete")
	err := s.repo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			log.Info("subscription not found")
		} else {
			log.Error("repository delete failed", zap.Error(err))
		}
		return err
	}

	log.Info("delete succeeded")
	return nil
}

func (s *ServiceImpl) List(ctx context.Context, userID, serviceName *string, limit, offset int) (ListResponse, error) {
	log := plog.FromContext(ctx).With(
		zap.String("op", "subscription.list"),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
	)

	if userID != nil {
		log = log.With(zap.String("user_id", strings.TrimSpace(*userID)))
	}
	if serviceName != nil {
		log = log.With(zap.String("service_name", strings.TrimSpace(*serviceName)))
	}

	log.Debug("list started")

	if limit <= 0 || limit > 200 {
		log.Warn("validation failed", zap.String("reason", "limit out of range"))
		return ListResponse{}, ErrInvalidInput
	}
	if offset < 0 {
		log.Warn("validation failed", zap.String("reason", "offset must be >= 0"))
		return ListResponse{}, ErrInvalidInput
	}

	log.Debug("calling repository list")
	items, total, err := s.repo.List(ctx, userID, serviceName, limit, offset)
	if err != nil {
		log.Error("repository list failed", zap.Error(err))
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

	log.Info("list succeeded", zap.Int("items", len(resp.Items)), zap.Int64("total", resp.Total))
	return resp, nil
}

func (s *ServiceImpl) Total(ctx context.Context, from, to string, userID, serviceName *string) (TotalResponse, error) {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)

	log := plog.FromContext(ctx).With(
		zap.String("op", "subscription.total"),
		zap.String("from", from),
		zap.String("to", to),
	)
	if userID != nil {
		log = log.With(zap.String("user_id", strings.TrimSpace(*userID)))
	}
	if serviceName != nil {
		log = log.With(zap.String("service_name", strings.TrimSpace(*serviceName)))
	}

	log.Debug("total started")

	fm, err := ParseMonthMMYYYY(from)
	if err != nil {
		log.Warn("validation failed", zap.String("reason", "invalid from"), zap.Error(err))
		return TotalResponse{}, ErrInvalidInput
	}
	tm, err := ParseMonthMMYYYY(to)
	if err != nil {
		log.Warn("validation failed", zap.String("reason", "invalid to"), zap.Error(err))
		return TotalResponse{}, ErrInvalidInput
	}
	if tm.Compare(fm) < 0 {
		log.Warn("validation failed", zap.String("reason", "to < from"))
		return TotalResponse{}, ErrInvalidInput
	}

	log.Debug("calling repository total")
	total, err := s.repo.Total(ctx, fm, tm, userID, serviceName)
	if err != nil {
		log.Error("repository total failed", zap.Error(err))
		return TotalResponse{}, err
	}

	resp := TotalResponse{
		From:        from,
		To:          to,
		UserID:      userID,
		ServiceName: serviceName,
		Total:       total,
	}

	log.Info("total succeeded", zap.Int64("total", total))
	return resp, nil
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

var _ Service = (*ServiceImpl)(nil)

func wrapNotFound(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	return err
}
