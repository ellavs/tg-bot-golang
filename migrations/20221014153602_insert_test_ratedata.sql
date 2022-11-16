-- +goose Up
-- +goose StatementBegin
-- Вставка тестовых курсов валют.
INSERT INTO exchangerates (currency, rate, period)
 VALUES ('USD', 0.015858969, CURRENT_TIMESTAMP),
        ('EUR', 0.0160078, CURRENT_TIMESTAMP),
        ('CNY', 0.11505467, CURRENT_TIMESTAMP)
 ON CONFLICT (currency, period) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM exchangerates
-- +goose StatementEnd