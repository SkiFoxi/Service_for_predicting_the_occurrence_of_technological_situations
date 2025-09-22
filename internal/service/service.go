package service

import (
    "context"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/google/uuid"
)

type ConsumptionAnalysis struct {
    BuildingID       uuid.UUID `json:"building_id"`
    Period           string    `json:"period"`
    TotalColdWater   int       `json:"total_cold_water"`
    TotalHotWater    int       `json:"total_hot_water"`
    Difference       int       `json:"difference"`
    DifferencePercent float64 `json:"difference_percent"`
    HasAnomalies     bool      `json:"has_anomalies"`
    AnomalyCount     int       `json:"anomaly_count"`
}

type Analyzer struct {
    pool *pgxpool.Pool
}

func NewAnalyzer(pool *pgxpool.Pool) *Analyzer {
    return &Analyzer{pool: pool}
}

func (a *Analyzer) AnalyzeConsumption(ctx context.Context, buildingID uuid.UUID, days int) (*ConsumptionAnalysis, error) {
    endDate := time.Now()
    startDate := endDate.AddDate(0, 0, -days)

    // Получаем данные по ХВС
    coldWater, err := a.getColdWaterData(ctx, buildingID, startDate, endDate)
    if err != nil {
        return nil, fmt.Errorf("get cold water data: %w", err)
    }

    // Получаем данные по ГВС
    hotWater, err := a.getHotWaterData(ctx, buildingID, startDate, endDate)
    if err != nil {
        return nil, fmt.Errorf("get hot water data: %w", err)
    }

    // Анализируем данные
    analysis := a.analyzeData(coldWater, hotWater, buildingID, startDate, endDate)
    
    return analysis, nil
}

func (a *Analyzer) getColdWaterData(ctx context.Context, buildingID uuid.UUID, start, end time.Time) ([]map[string]interface{}, error) {
    rows, err := a.pool.Query(ctx, `
        SELECT cwm.flow_rate, cwm.timestamp 
        FROM cold_water_meters cwm
        JOIN itp i ON cwm.itp_id = i.id
        WHERE i.building_id = $1 AND cwm.timestamp BETWEEN $2 AND $3
        ORDER BY cwm.timestamp`, buildingID, start, end)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var data []map[string]interface{}
    for rows.Next() {
        var flowRate int
        var timestamp time.Time
        err := rows.Scan(&flowRate, &timestamp)
        if err != nil {
            return nil, err
        }
        data = append(data, map[string]interface{}{
            "flow_rate": flowRate,
            "timestamp": timestamp,
        })
    }

    return data, nil
}

func (a *Analyzer) getHotWaterData(ctx context.Context, buildingID uuid.UUID, start, end time.Time) ([]map[string]interface{}, error) {
    rows, err := a.pool.Query(ctx, `
        SELECT flow_rate_ch1, flow_rate_ch2, timestamp 
        FROM hot_water_meters 
        WHERE building_id = $1 AND timestamp BETWEEN $2 AND $3
        ORDER BY timestamp`, buildingID, start, end)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var data []map[string]interface{}
    for rows.Next() {
        var flowRateCh1, flowRateCh2 int
        var timestamp time.Time
        err := rows.Scan(&flowRateCh1, &flowRateCh2, &timestamp)
        if err != nil {
            return nil, err
        }
        data = append(data, map[string]interface{}{
            "flow_rate_ch1": flowRateCh1,
            "flow_rate_ch2": flowRateCh2,
            "total_flow":    flowRateCh1 + flowRateCh2,
            "timestamp":     timestamp,
        })
    }

    return data, nil
}

func (a *Analyzer) analyzeData(coldWater, hotWater []map[string]interface{}, buildingID uuid.UUID, start, end time.Time) *ConsumptionAnalysis {
    // Здесь реализуйте логику анализа
    // Вычисление разниц, поиск аномалий и т.д.
    
    return &ConsumptionAnalysis{
        BuildingID:       buildingID,
        Period:           fmt.Sprintf("%s to %s", start.Format("2006-01-02"), end.Format("2006-01-02")),
        TotalColdWater:   1000, // Примерные значения
        TotalHotWater:    800,
        Difference:       200,
        DifferencePercent: 20.0,
        HasAnomalies:     true,
        AnomalyCount:     3,
    }
}