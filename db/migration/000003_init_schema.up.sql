-- Удаляем старые таблицы (не соответствующие ТЗ)
DROP TABLE IF EXISTS "accounts";
DROP TABLE IF EXISTS "transfers";

-- Переименовываем существующую таблицу (временно)
ALTER TABLE IF EXISTS "waterMeterStatement" RENAME TO old_water_meter_statement;

-- Создаем новые таблицы согласно ТЗ

-- Таблица зданий (МКД)
CREATE TABLE buildings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    address TEXT NOT NULL,
    fias_id TEXT UNIQUE,
    unom_id TEXT UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Таблица ИТП
CREATE TABLE itp (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    itp_number TEXT NOT NULL,
    building_id UUID NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Таблица счетчиков ХВС в ИТП
CREATE TABLE cold_water_meters (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    itp_id UUID NOT NULL REFERENCES itp(id) ON DELETE CASCADE,
    flow_rate INTEGER NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Уникальность показаний для конкретного ИТП и времени
    UNIQUE(itp_id, timestamp)
);

-- Таблица ОДПУ ГВС в МКД
CREATE TABLE hot_water_meters (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    building_id UUID NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
    flow_rate_ch1 INTEGER NOT NULL,  -- канал 1
    flow_rate_ch2 INTEGER NOT NULL,  -- канал 2
    timestamp TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Уникальность показаний для конкретного здания и времени
    UNIQUE(building_id, timestamp)
);

-- Создаем индексы для ускорения запросов
CREATE INDEX idx_cold_water_meters_timestamp ON cold_water_meters(timestamp);
CREATE INDEX idx_hot_water_meters_timestamp ON hot_water_meters(timestamp);
CREATE INDEX idx_cold_water_meters_itp_id ON cold_water_meters(itp_id);
CREATE INDEX idx_hot_water_meters_building_id ON hot_water_meters(building_id);