package subscription

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	plog "subs-service/internal/platform/logger"
)

type RepositoryPG struct {
	pool *pgxpool.Pool
}

func NewRepositoryPG(pool *pgxpool.Pool) *RepositoryPG {
	return &RepositoryPG{pool: pool}
}

func (r *RepositoryPG) Create(ctx context.Context, s Subscription) (Subscription, error) {
	log := plog.FromContext(ctx).With(
		zap.String("op", "repo.subscription.create"),
		zap.String("user_id", s.UserID),
		zap.String("service_name", s.ServiceName),
		zap.Int64("price", s.Price),
		zap.String("start_month", s.StartMonth.String()),
		zap.Bool("has_end_month", s.EndMonth != nil),
	)

	const q = `
INSERT INTO subscriptions (service_name, price, user_id, start_month, end_month)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, service_name, price, user_id, start_month, end_month, created_at, updated_at;
`

	var end any = nil
	if s.EndMonth != nil {
		end = s.EndMonth.Time()
		log = log.With(zap.String("end_month", s.EndMonth.String()))
	}

	log.Debug("db insert started")

	row := r.pool.QueryRow(ctx, q,
		s.ServiceName,
		s.Price,
		s.UserID,
		s.StartMonth.Time(),
		end,
	)

	var start time.Time
	var endT *time.Time
	err := row.Scan(&s.ID, &s.ServiceName, &s.Price, &s.UserID, &start, &endT, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		log.Error("db insert failed", zap.Error(err))
		return Subscription{}, classifyPGError(err)
	}

	s.StartMonth = Month{t: start}
	if endT != nil {
		m := Month{t: *endT}
		s.EndMonth = &m
	} else {
		s.EndMonth = nil
	}

	log.Debug("db insert succeeded", zap.String("subscription_id", s.ID))
	return s, nil
}

func (r *RepositoryPG) Get(ctx context.Context, id string) (Subscription, error) {
	log := plog.FromContext(ctx).With(
		zap.String("op", "repo.subscription.get"),
		zap.String("subscription_id", id),
	)

	const q = `
SELECT id, service_name, price, user_id, start_month, end_month, created_at, updated_at
FROM subscriptions
WHERE id = $1;
`

	log.Debug("db select started")

	var s Subscription
	var start time.Time
	var endT *time.Time

	err := r.pool.QueryRow(ctx, q, id).Scan(
		&s.ID, &s.ServiceName, &s.Price, &s.UserID,
		&start, &endT, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Debug("db select no rows")
			return Subscription{}, ErrNotFound
		}
		log.Error("db select failed", zap.Error(err))
		return Subscription{}, classifyPGError(err)
	}

	s.StartMonth = Month{t: start}
	if endT != nil {
		m := Month{t: *endT}
		s.EndMonth = &m
	}

	log.Debug("db select succeeded",
		zap.String("user_id", s.UserID),
		zap.String("service_name", s.ServiceName),
	)
	return s, nil
}

func (r *RepositoryPG) Update(ctx context.Context, s Subscription) (Subscription, error) {
	log := plog.FromContext(ctx).With(
		zap.String("op", "repo.subscription.update"),
		zap.String("subscription_id", s.ID),
		zap.String("user_id", s.UserID),
		zap.String("service_name", s.ServiceName),
		zap.Int64("price", s.Price),
		zap.String("start_month", s.StartMonth.String()),
		zap.Bool("has_end_month", s.EndMonth != nil),
	)

	const q = `
UPDATE subscriptions
SET service_name = $2,
    price = $3,
    user_id = $4,
    start_month = $5,
    end_month = $6,
    updated_at = now()
WHERE id = $1
RETURNING id, service_name, price, user_id, start_month, end_month, created_at, updated_at;
`

	var end any = nil
	if s.EndMonth != nil {
		end = s.EndMonth.Time()
		log = log.With(zap.String("end_month", s.EndMonth.String()))
	}

	log.Debug("db update started")

	var start time.Time
	var endT *time.Time
	err := r.pool.QueryRow(ctx, q,
		s.ID, s.ServiceName, s.Price, s.UserID, s.StartMonth.Time(), end,
	).Scan(&s.ID, &s.ServiceName, &s.Price, &s.UserID, &start, &endT, &s.CreatedAt, &s.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Debug("db update no rows")
			return Subscription{}, ErrNotFound
		}
		log.Error("db update failed", zap.Error(err))
		return Subscription{}, classifyPGError(err)
	}

	s.StartMonth = Month{t: start}
	if endT != nil {
		m := Month{t: *endT}
		s.EndMonth = &m
	} else {
		s.EndMonth = nil
	}

	log.Debug("db update succeeded")
	return s, nil
}

func (r *RepositoryPG) Delete(ctx context.Context, id string) error {
	log := plog.FromContext(ctx).With(
		zap.String("op", "repo.subscription.delete"),
		zap.String("subscription_id", id),
	)

	const q = `DELETE FROM subscriptions WHERE id = $1;`

	log.Debug("db delete started")

	ct, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		log.Error("db delete failed", zap.Error(err))
		return classifyPGError(err)
	}
	if ct.RowsAffected() == 0 {
		log.Debug("db delete no rows")
		return ErrNotFound
	}

	log.Debug("db delete succeeded", zap.Int64("rows_affected", ct.RowsAffected()))
	return nil
}

func (r *RepositoryPG) List(ctx context.Context, userID, serviceName *string, limit, offset int) ([]Subscription, int64, error) {
	log := plog.FromContext(ctx).With(
		zap.String("op", "repo.subscription.list"),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
	)

	const q = `
SELECT id, service_name, price, user_id, start_month, end_month, created_at, updated_at
FROM subscriptions
WHERE ($1::uuid IS NULL OR user_id = $1::uuid)
  AND ($2::text IS NULL OR service_name = $2::text)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;
`
	const qc = `
SELECT COUNT(*)
FROM subscriptions
WHERE ($1::uuid IS NULL OR user_id = $1::uuid)
  AND ($2::text IS NULL OR service_name = $2::text);
`

	var uid any = nil
	if userID != nil {
		uid = *userID
		log = log.With(zap.String("user_id", *userID))
	}
	var sn any = nil
	if serviceName != nil {
		sn = *serviceName
		log = log.With(zap.String("service_name", *serviceName))
	}

	log.Debug("db count started")
	var total int64
	if err := r.pool.QueryRow(ctx, qc, uid, sn).Scan(&total); err != nil {
		log.Error("db count failed", zap.Error(err))
		return nil, 0, classifyPGError(err)
	}
	log.Debug("db count succeeded", zap.Int64("total", total))

	log.Debug("db list started")
	rows, err := r.pool.Query(ctx, q, uid, sn, limit, offset)
	if err != nil {
		log.Error("db list query failed", zap.Error(err))
		return nil, 0, classifyPGError(err)
	}
	defer rows.Close()

	items := make([]Subscription, 0, limit)
	for rows.Next() {
		var s Subscription
		var start time.Time
		var endT *time.Time
		if err := rows.Scan(&s.ID, &s.ServiceName, &s.Price, &s.UserID, &start, &endT, &s.CreatedAt, &s.UpdatedAt); err != nil {
			log.Error("db list scan failed", zap.Error(err))
			return nil, 0, classifyPGError(err)
		}
		s.StartMonth = Month{t: start}
		if endT != nil {
			m := Month{t: *endT}
			s.EndMonth = &m
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		log.Error("db list rows error", zap.Error(err))
		return nil, 0, classifyPGError(err)
	}

	log.Debug("db list succeeded", zap.Int("items", len(items)))
	return items, total, nil
}

func (r *RepositoryPG) Total(ctx context.Context, from Month, to Month, userID, serviceName *string) (int64, error) {
	log := plog.FromContext(ctx).With(
		zap.String("op", "repo.subscription.total"),
		zap.String("from_month", from.String()),
		zap.String("to_month", to.String()),
	)

	const q = `
WITH params AS (
  SELECT
    date_trunc('month', $1::date)::date AS from_m,
    date_trunc('month', $2::date)::date AS to_m,
    $3::uuid AS user_id,
    $4::text AS service_name
),
months AS (
  SELECT generate_series((SELECT from_m FROM params), (SELECT to_m FROM params), interval '1 month')::date AS m
)
SELECT COALESCE(SUM(s.price), 0)::bigint AS total
FROM months
JOIN subscriptions s
  ON s.start_month <= months.m
 AND (s.end_month IS NULL OR s.end_month >= months.m)
WHERE
  ((SELECT user_id FROM params) IS NULL OR s.user_id = (SELECT user_id FROM params))
  AND ((SELECT service_name FROM params) IS NULL OR s.service_name = (SELECT service_name FROM params));
`

	var uid any = nil
	if userID != nil {
		uid = *userID
		log = log.With(zap.String("user_id", *userID))
	}
	var sn any = nil
	if serviceName != nil {
		sn = *serviceName
		log = log.With(zap.String("service_name", *serviceName))
	}

	log.Debug("db total started")

	var total int64
	if err := r.pool.QueryRow(ctx, q, from.Time(), to.Time(), uid, sn).Scan(&total); err != nil {
		log.Error("db total failed", zap.Error(err))
		return 0, classifyPGError(err)
	}

	log.Debug("db total succeeded", zap.Int64("total", total))
	return total, nil
}

var _ Repository = (*RepositoryPG)(nil)
