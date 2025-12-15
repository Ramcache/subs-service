package subscription

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryPG struct {
	pool *pgxpool.Pool
}

func NewRepositoryPG(pool *pgxpool.Pool) *RepositoryPG {
	return &RepositoryPG{pool: pool}
}

func (r *RepositoryPG) Create(ctx context.Context, s Subscription) (Subscription, error) {
	const q = `
INSERT INTO subscriptions (service_name, price, user_id, start_month, end_month)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, service_name, price, user_id, start_month, end_month, created_at, updated_at;
`
	var end any = nil
	if s.EndMonth != nil {
		end = s.EndMonth.Time()
	}

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
		return Subscription{}, err
	}
	s.StartMonth = Month{t: start}
	if endT != nil {
		m := Month{t: *endT}
		s.EndMonth = &m
	} else {
		s.EndMonth = nil
	}
	return s, nil
}

func (r *RepositoryPG) Get(ctx context.Context, id string) (Subscription, error) {
	const q = `
SELECT id, service_name, price, user_id, start_month, end_month, created_at, updated_at
FROM subscriptions
WHERE id = $1;
`
	var s Subscription
	var start time.Time
	var endT *time.Time

	err := r.pool.QueryRow(ctx, q, id).Scan(
		&s.ID, &s.ServiceName, &s.Price, &s.UserID,
		&start, &endT, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Subscription{}, ErrNotFound
		}
		return Subscription{}, err
	}

	s.StartMonth = Month{t: start}
	if endT != nil {
		m := Month{t: *endT}
		s.EndMonth = &m
	}
	return s, nil
}

func (r *RepositoryPG) Update(ctx context.Context, s Subscription) (Subscription, error) {
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
	}

	var start time.Time
	var endT *time.Time
	err := r.pool.QueryRow(ctx, q,
		s.ID, s.ServiceName, s.Price, s.UserID, s.StartMonth.Time(), end,
	).Scan(&s.ID, &s.ServiceName, &s.Price, &s.UserID, &start, &endT, &s.CreatedAt, &s.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Subscription{}, ErrNotFound
		}
		return Subscription{}, err
	}

	s.StartMonth = Month{t: start}
	if endT != nil {
		m := Month{t: *endT}
		s.EndMonth = &m
	} else {
		s.EndMonth = nil
	}
	return s, nil
}

func (r *RepositoryPG) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM subscriptions WHERE id = $1;`
	ct, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *RepositoryPG) List(ctx context.Context, userID, serviceName *string, limit, offset int) ([]Subscription, int64, error) {
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
	}
	var sn any = nil
	if serviceName != nil {
		sn = *serviceName
	}

	var total int64
	if err := r.pool.QueryRow(ctx, qc, uid, sn).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, q, uid, sn, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]Subscription, 0, limit)
	for rows.Next() {
		var s Subscription
		var start time.Time
		var endT *time.Time
		if err := rows.Scan(&s.ID, &s.ServiceName, &s.Price, &s.UserID, &start, &endT, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, 0, err
		}
		s.StartMonth = Month{t: start}
		if endT != nil {
			m := Month{t: *endT}
			s.EndMonth = &m
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *RepositoryPG) Total(ctx context.Context, from Month, to Month, userID, serviceName *string) (int64, error) {
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
	}
	var sn any = nil
	if serviceName != nil {
		sn = *serviceName
	}

	var total int64
	if err := r.pool.QueryRow(ctx, q, from.Time(), to.Time(), uid, sn).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

var _ Repository = (*RepositoryPG)(nil)
