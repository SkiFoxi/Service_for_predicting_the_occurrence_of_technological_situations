-- Заполнение первоначальных данных для тестирования

-- 1. Создаем тестовые здания
INSERT INTO buildings (id, address, fias_id, unom_id, created_at, updated_at) VALUES
('11111111-1111-1111-1111-111111111111', 'г. Москва, ул. Ленина, д. 10', 'fias-001', 'unom-1001', NOW(), NOW()),
('22222222-2222-2222-2222-222222222222', 'г. Москва, пр. Мира, д. 25', 'fias-002', 'unom-1002', NOW(), NOW()),
('33333333-3333-3333-3333-333333333333', 'г. Москва, ул. Гагарина, д. 15', 'fias-003', 'unom-1003', NOW(), NOW());

-- 2. Создаем ИТП для каждого здания
INSERT INTO itp (id, itp_number, building_id, created_at, updated_at) VALUES
(uuid_generate_v4(), 'ИТП-001', '11111111-1111-1111-1111-111111111111', NOW(), NOW()),
(uuid_generate_v4(), 'ИТП-002', '22222222-2222-2222-2222-222222222222', NOW(), NOW()),
(uuid_generate_v4(), 'ИТП-003', '33333333-3333-3333-3333-333333333333', NOW(), NOW());

-- 3. Заполняем данными за последние 7 дней
DO $$
DECLARE
    building_record RECORD;
    itp_record RECORD;
    current_time TIMESTAMPTZ;
    i INTEGER;
    j INTEGER;
BEGIN
    FOR building_record IN SELECT id FROM buildings LOOP
        -- Получаем ITP для здания
        SELECT id INTO itp_record FROM itp WHERE building_id = building_record.id LIMIT 1;
        
        FOR i IN 0..6 LOOP  -- 7 дней
            current_time := NOW() - (i * INTERVAL '1 day');
            
            FOR j IN 0..23 LOOP  -- 24 часа в сутки
                current_time := current_time + (j * INTERVAL '1 hour');
                
                -- Данные ГВС
                INSERT INTO hot_water_meters (id, building_id, flow_rate_ch1, flow_rate_ch2, timestamp, created_at)
                VALUES (
                    uuid_generate_v4(),
                    building_record.id,
                    2 + random() * 4,  -- 2-6 м³/ч
                    1 + random() * 3,  -- 1-4 м³/ч
                    current_time,
                    NOW()
                );
                
                -- Данные ХВС
                INSERT INTO cold_water_meters (id, itp_id, flow_rate, timestamp, created_at)
                VALUES (
                    uuid_generate_v4(),
                    itp_record.id,
                    3 + random() * 7,  -- 3-10 м³/ч
                    current_time,
                    NOW()
                );
                
                -- Температурные данные (раз в 6 часов)
                IF j % 6 = 0 THEN
                    INSERT INTO temperature_readings (id, building_id, supply_temp, return_temp, delta_temp, timestamp, created_at)
                    VALUES (
                        uuid_generate_v4(),
                        building_record.id,
                        60 + random() * 10,  -- 60-70°C
                        40 + random() * 10,  -- 40-50°C
                        20,  -- стабильная разница
                        current_time,
                        NOW()
                    );
                END IF;
            END LOOP;
            
            -- Данные насосов (раз в день)
            INSERT INTO pump_data (id, building_id, pump_number, status, operating_hours, 
                                 pressure_input, pressure_output, vibration_level, timestamp, created_at)
            VALUES 
            (
                uuid_generate_v4(),
                building_record.id,
                'Pump-1',
                CASE WHEN random() < 0.9 THEN 'normal' ELSE 'warning' END,
                5000 + (i * 24) + random() * 1000,
                2 + random() * 2,
                4 + random() * 2,
                random() * 5,
                current_time,
                NOW()
            ),
            (
                uuid_generate_v4(),
                building_record.id,
                'Pump-2',
                CASE WHEN random() < 0.95 THEN 'normal' ELSE 'warning' END,
                3000 + (i * 24) + random() * 1000,
                2 + random() * 2,
                4 + random() * 2,
                random() * 5,
                current_time,
                NOW()
            );
        END LOOP;
    END LOOP;
END $$;

-- Проверяем заполнение
SELECT 
    (SELECT COUNT(*) FROM buildings) as buildings_count,
    (SELECT COUNT(*) FROM itp) as itp_count,
    (SELECT COUNT(*) FROM cold_water_meters) as cold_water_records,
    (SELECT COUNT(*) FROM hot_water_meters) as hot_water_records,
    (SELECT COUNT(*) FROM temperature_readings) as temperature_records,
    (SELECT COUNT(*) FROM pump_data) as pump_records;