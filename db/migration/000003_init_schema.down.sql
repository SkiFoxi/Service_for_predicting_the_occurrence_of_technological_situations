-- Восстанавливаем исходное состояние
DROP TABLE IF EXISTS hot_water_meters;
DROP TABLE IF EXISTS cold_water_meters;
DROP TABLE IF EXISTS itp;
DROP TABLE IF EXISTS buildings;
DROP TABLE IF EXISTS temperature_readings;
DROP TABLE IF EXISTS pump_data;
-- Восстанавливаем старую таблицу (если нужно)
ALTER TABLE IF EXISTS old_water_meter_statement RENAME TO "waterMeterStatement";