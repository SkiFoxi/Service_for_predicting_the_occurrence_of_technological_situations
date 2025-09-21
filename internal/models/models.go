package models

import (
    "time"
    "github.com/google/uuid" //генерирует ключи uuid
)
//Тип таблицы строений МКД (Многоквартирный дом)
type Building struct {
    ID        uuid.UUID `json:"id" db:"id"`
    Address   string    `json:"address" db:"address"`
    FiasID    string    `json:"fias_id" db:"fias_id"`  // ФИАС-код (федеральная система адресов)
    UnomID    string    `json:"unom_id" db:"unom_id"`  //УНОМ-код (уникальный номер объекта недвижимости)
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`  //время последнего обновления
}
//Структура ITP Индивидуального Теплового Пункта
type ColdWaterMeter struct {
    ID        uuid.UUID `json:"id" db:"id"`
    ITPID     uuid.UUID `json:"itp_id" db:"itp_id"`  //номер ИТП
    FlowRate  int       `json:"flow_rate" db:"flow_rate"` //ссылка на здание
    Timestamp time.Time `json:"timestamp" db:"timestamp"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}
//Структура Счетчика ХВС
type HotWaterMeter struct {
    ID           uuid.UUID `json:"id" db:"id"`
    BuildingID   uuid.UUID `json:"building_id" db:"building_id"` //ссылка на здание
    FlowRateCh1  int       `json:"flow_rate_ch1" db:"flow_rate_ch1"` //Канал 1
    FlowRateCh2  int       `json:"flow_rate_ch2" db:"flow_rate_ch2"` // Канал 2
    Timestamp    time.Time `json:"timestamp" db:"timestamp"`   // время измерения
    CreatedAt    time.Time `json:"created_at" db:"created_at"`
}