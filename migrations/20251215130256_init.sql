-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS subscriptions (
                                             id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
                                             service_name text NOT NULL,
                                             price bigint NOT NULL CHECK (price >= 0),
                                             user_id uuid NOT NULL,
                                             start_month date NOT NULL,
                                             end_month date NULL,
                                             created_at timestamptz NOT NULL DEFAULT now(),
                                             updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_service_name ON subscriptions(service_name);
CREATE INDEX IF NOT EXISTS idx_subscriptions_period ON subscriptions(start_month, end_month);

-- +goose Down
DROP TABLE IF EXISTS subscriptions;
