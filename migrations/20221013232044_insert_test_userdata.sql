-- +goose Up
-- +goose StatementBegin
DO
$$
    DECLARE
        testUserId integer;
        recs       RECORD;
    BEGIN
        -- Вставка тестового пользователя.
        INSERT INTO users
        (tg_id, name, currency, limits)
        VALUES (1234567890, 'ellavs', 'RUB', 50000)
        ON CONFLICT (tg_id) DO NOTHING;

        SELECT id FROM users WHERE tg_id = 1234567890 INTO testUserId;

        -- Вставка тестовых категорий пользователя.
        INSERT INTO usercategories (user_id, name)
        VALUES (testUserId, 'Бензин'),
               (testUserId, 'Продукты'),
               (testUserId, 'Здоровье'),
               (testUserId, 'Обувь'),
               (testUserId, 'Интернет')
        ON CONFLICT (user_id, lower(name)) DO NOTHING;

        -- Вставка тестовых расходов пользователя.
        FOR recs IN
            SELECT id, name
            FROM usercategories
            WHERE user_id = testUserId
            LOOP
                CASE recs.name
                    WHEN 'Бензин' THEN INSERT INTO usermoneytransactions (user_id, category_id, period, sum)
                                       VALUES (testUserId, recs.id, CURRENT_TIMESTAMP, 50000),
                                              (testUserId, recs.id, CURRENT_TIMESTAMP - interval '20 hour', 150000),
                                              (testUserId, recs.id, CURRENT_TIMESTAMP - interval '2 days', 55000),
                                              (testUserId, recs.id, CURRENT_TIMESTAMP - interval '10 days', 45000),
                                              (testUserId, recs.id, CURRENT_TIMESTAMP - interval '1 month 1 days', 135000),
                                              -- Тестовая запись с периодом, выходящим за пределы последнего года
                                              (testUserId, recs.id, CURRENT_TIMESTAMP - interval '2 years', 60000)
                                       ON CONFLICT DO NOTHING;
                    WHEN 'Продукты' THEN INSERT INTO usermoneytransactions (user_id, category_id, period, sum)
                                         VALUES (testUserId, recs.id, CURRENT_TIMESTAMP - interval '2 hour', 253555),
                                                (testUserId, recs.id, CURRENT_TIMESTAMP - interval '32 hour', 85600),
                                                (testUserId, recs.id, CURRENT_TIMESTAMP - interval '1 days', 74500),
                                                (testUserId, recs.id, CURRENT_TIMESTAMP - interval '32 days', 95230),
                                                (testUserId, recs.id, CURRENT_TIMESTAMP - interval '36 days', 345000),
                                                (testUserId, recs.id, CURRENT_TIMESTAMP - interval '3 month', 156000),
                                                -- Тестовая запись с периодом, выходящим за пределы последнего года
                                                (testUserId, recs.id, CURRENT_TIMESTAMP - interval '2 years', 85000)
                                         ON CONFLICT DO NOTHING;
                    WHEN 'Здоровье' THEN INSERT INTO usermoneytransactions (user_id, category_id, period, sum)
                                         VALUES (testUserId, recs.id, CURRENT_TIMESTAMP - interval '1 days', 1562000),
                                                (testUserId, recs.id, CURRENT_TIMESTAMP - interval '33 days', 860000),
                                                (testUserId, recs.id, CURRENT_TIMESTAMP - interval '37 days', 530000),
                                                (testUserId, recs.id, CURRENT_TIMESTAMP - interval '2 month', 2300000),
                                                -- Тестовая запись с периодом, выходящим за пределы последнего года
                                                (testUserId, recs.id, CURRENT_TIMESTAMP - interval '2 years', 1500000)
                                         ON CONFLICT DO NOTHING;
                    WHEN 'Обувь' THEN INSERT INTO usermoneytransactions (user_id, category_id, period, sum)
                                      VALUES (testUserId, recs.id, CURRENT_TIMESTAMP - interval '3 days', 890000),
                                             (testUserId, recs.id, CURRENT_TIMESTAMP - interval '38 days', 260000),
                                             (testUserId, recs.id, CURRENT_TIMESTAMP - interval '5 month', 1300000),
                                             -- Тестовая запись с периодом, выходящим за пределы последнего года
                                             (testUserId, recs.id, CURRENT_TIMESTAMP - interval '2 years', 120000)
                                      ON CONFLICT DO NOTHING;
                    WHEN 'Интернет' THEN INSERT INTO usermoneytransactions (user_id, category_id, period, sum)
                                         VALUES (testUserId, recs.id, CURRENT_TIMESTAMP - interval '20 days', 96000),
                                                (testUserId, recs.id, CURRENT_TIMESTAMP - interval '2 month', 96000),
                                                (testUserId, recs.id, CURRENT_TIMESTAMP - interval '3 month', 96000),
                                                (testUserId, recs.id, CURRENT_TIMESTAMP - interval '4 month', 95000),
                                                (testUserId, recs.id, CURRENT_TIMESTAMP - interval '5 month', 95000),
                                                -- Тестовая запись с периодом, выходящим за пределы последнего года
                                                (testUserId, recs.id, CURRENT_TIMESTAMP - interval '2 years', 90000)
                                         ON CONFLICT DO NOTHING;
                    END CASE;
            END LOOP;
    END
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Удаление остальных данных произойдет каскадно.
DELETE FROM users WHERE tg_id = 1234567890
-- +goose StatementEnd