-- Восстанавливаем исходное состояние
DROP TABLE IF EXISTS hot_water_meters;
DROP TABLE IF EXISTS cold_water_meters;
DROP TABLE IF EXISTS itp;
DROP TABLE IF EXISTS buildings;

-- Восстанавливаем старую таблицу (если нужно)
ALTER TABLE IF EXISTS old_water_meter_statement RENAME TO "waterMeterStatement";