package models

import (
    "time"
    "github.com/google/uuid"
)

// Тип таблицы строений МКД (Многоквартирный дом)
type Building struct {
    ID        uuid.UUID `json:"id" db:"id"`
    Address   string    `json:"address" db:"address"`
    FiasID    string    `json:"fias_id" db:"fias_id"`
    UnomID    string    `json:"unom_id" db:"unom_id"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Структура ITP Индивидуального Теплового Пункта
type ITP struct {
    ID         uuid.UUID `json:"id" db:"id"`
    ITPNumber  string    `json:"itp_number" db:"itp_number"`
    BuildingID uuid.UUID `json:"building_id" db:"building_id"`
    CreatedAt  time.Time `json:"created_at" db:"created_at"`
    UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// Структура Счетчика ХВС
type ColdWaterMeter struct {
    ID        uuid.UUID `json:"id" db:"id"`
    ITPID     uuid.UUID `json:"itp_id" db:"itp_id"`
    FlowRate  int       `json:"flow_rate" db:"flow_rate"`
    Timestamp time.Time `json:"timestamp" db:"timestamp"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Структура Счетчика ГВС
type HotWaterMeter struct {
    ID          uuid.UUID `json:"id" db:"id"`
    BuildingID  uuid.UUID `json:"building_id" db:"building_id"`
    FlowRateCh1 int       `json:"flow_rate_ch1" db:"flow_rate_ch1"`
    FlowRateCh2 int       `json:"flow_rate_ch2" db:"flow_rate_ch2"`
    Timestamp   time.Time `json:"timestamp" db:"timestamp"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Структура температурных данных
type TemperatureReading struct {
    ID         uuid.UUID `json:"id" db:"id"`
    BuildingID uuid.UUID `json:"building_id" db:"building_id"`
    SupplyTemp int       `json:"supply_temp" db:"supply_temp"`   // температура подачи
    ReturnTemp int       `json:"return_temp" db:"return_temp"`   // температура возврата
    DeltaTemp  int       `json:"delta_temp" db:"delta_temp"`     // разница температур
    Timestamp  time.Time `json:"timestamp" db:"timestamp"`
    CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// Структура данных насосов
type PumpData struct {
    ID             uuid.UUID `json:"id" db:"id"`
    BuildingID     uuid.UUID `json:"building_id" db:"building_id"`
    PumpNumber     string    `json:"pump_number" db:"pump_number"`
    Status         string    `json:"status" db:"status"`
    OperatingHours int       `json:"operating_hours" db:"operating_hours"`
    PressureInput  int       `json:"pressure_input" db:"pressure_input"`
    PressureOutput int       `json:"pressure_output" db:"pressure_output"`
    VibrationLevel int       `json:"vibration_level" db:"vibration_level"`
    Timestamp      time.Time `json:"timestamp" db:"timestamp"`
    CreatedAt      time.Time `json:"created_at" db:"created_at"`
}