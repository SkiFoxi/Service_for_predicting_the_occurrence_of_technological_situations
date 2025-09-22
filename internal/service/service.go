package service

import (
    "context"
    "fmt"
    "math/rand"
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
    // Расчет суммарного потребления
    totalColdWater := 0
    for _, data := range coldWater {
        if flowRate, ok := data["flow_rate"].(int); ok {
            totalColdWater += flowRate
        }
    }

    totalHotWater := 0
    for _, data := range hotWater {
        if totalFlow, ok := data["total_flow"].(int); ok {
            totalHotWater += totalFlow
        }
    }

    // Расчет разницы и процента
    difference := totalColdWater - totalHotWater
    var differencePercent float64
    if totalColdWater > 0 {
        differencePercent = (float64(difference) / float64(totalColdWater)) * 100
    }

    // Простой детектор аномалий (значения > 150 или < 5 считаем аномалиями)
    anomalyCount := 0
    for _, data := range coldWater {
        if flowRate, ok := data["flow_rate"].(int); ok {
            if flowRate > 150 || flowRate < 5 {
                anomalyCount++
            }
        }
    }

    for _, data := range hotWater {
        if totalFlow, ok := data["total_flow"].(int); ok {
            if totalFlow > 150 || totalFlow < 5 {
                anomalyCount++
            }
        }
    }

    return &ConsumptionAnalysis{
        BuildingID:       buildingID,
        Period:           fmt.Sprintf("%s to %s", start.Format("2006-01-02"), end.Format("2006-01-02")),
        TotalColdWater:   totalColdWater,
        TotalHotWater:    totalHotWater,
        Difference:       difference,
        DifferencePercent: differencePercent,
        HasAnomalies:     anomalyCount > 0,
        AnomalyCount:     anomalyCount,
    }
}

// DataGenerator для генерации тестовых данных в реальном времени
type DataGenerator struct {
    pool        *pgxpool.Pool
    buildings   []uuid.UUID
    itps        []uuid.UUID
    stopChan    chan bool
    isRunning   bool
}

func NewDataGenerator(pool *pgxpool.Pool) *DataGenerator {
    return &DataGenerator{
        pool:     pool,
        stopChan: make(chan bool),
    }
}

func (dg *DataGenerator) LoadExistingData(ctx context.Context) error {
    // Загружаем здания
    rows, err := dg.pool.Query(ctx, "SELECT id FROM buildings")
    if err != nil {
        return err
    }
    defer rows.Close()

    for rows.Next() {
        var id uuid.UUID
        if err := rows.Scan(&id); err != nil {
            return err
        }
        dg.buildings = append(dg.buildings, id)
    }

    // Загружаем ИТП
    rows, err = dg.pool.Query(ctx, "SELECT id FROM itp")
    if err != nil {
        return err
    }
    defer rows.Close()

    for rows.Next() {
        var id uuid.UUID
        if err := rows.Scan(&id); err != nil {
            return err
        }
        dg.itps = append(dg.itps, id)
    }

    return nil
}

func (dg *DataGenerator) Start(ctx context.Context) {
    if dg.isRunning {
        return
    }
    dg.isRunning = true

    go dg.generateColdWaterData(ctx)
    go dg.generateHotWaterData(ctx)
    go dg.generateAnomalies(ctx)

    fmt.Println("Data generation started")
}

func (dg *DataGenerator) Stop() {
    if !dg.isRunning {
        return
    }
    
    dg.stopChan <- true
    dg.isRunning = false
    fmt.Println("Data generation stopped")
}

func (dg *DataGenerator) IsRunning() bool {
    return dg.isRunning
}

func (dg *DataGenerator) generateColdWaterData(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second) // Каждые 30 секунд для демо
    defer ticker.Stop()

    for {
        select {
        case <-dg.stopChan:
            return
        case <-ticker.C:
            for _, itpID := range dg.itps {
                baseFlow := 50 + rand.Intn(50) // 50-100 м³/ч
                fluctuation := rand.Intn(20) - 10 // ±10 колебание
                flowRate := baseFlow + fluctuation
                
                if flowRate < 0 {
                    flowRate = 5
                }

                _, err := dg.pool.Exec(ctx, `
                    INSERT INTO cold_water_meters (id, itp_id, flow_rate, timestamp, created_at)
                    VALUES ($1, $2, $3, $4, NOW())`,
                    uuid.New(), itpID, flowRate, time.Now())
                
                if err != nil {
                    fmt.Printf("Error inserting cold water data: %v\n", err)
                }
            }
        }
    }
}

func (dg *DataGenerator) generateHotWaterData(ctx context.Context) {
    ticker := time.NewTicker(45 * time.Second) // Каждые 45 секунд для демо
    defer ticker.Stop()

    for {
        select {
        case <-dg.stopChan:
            return
        case <-ticker.C:
            for _, buildingID := range dg.buildings {
                flowRateCh1 := 20 + rand.Intn(30) // 20-50 м³/ч
                flowRateCh2 := 10 + rand.Intn(20) // 10-30 м³/ч

                _, err := dg.pool.Exec(ctx, `
                    INSERT INTO hot_water_meters (id, building_id, flow_rate_ch1, flow_rate_ch2, timestamp, created_at)
                    VALUES ($1, $2, $3, $4, $5, NOW())`,
                    uuid.New(), buildingID, flowRateCh1, flowRateCh2, time.Now())
                
                if err != nil {
                    fmt.Printf("Error inserting hot water data: %v\n", err)
                }
            }
        }
    }
}

func (dg *DataGenerator) generateAnomalies(ctx context.Context) {
    ticker := time.NewTicker(2 * time.Minute) // Каждые 2 минуты
    defer ticker.Stop()

    for {
        select {
        case <-dg.stopChan:
            return
        case <-ticker.C:
            if rand.Float32() < 0.3 && len(dg.itps) > 0 { // 30% chance of anomaly
                itpID := dg.itps[rand.Intn(len(dg.itps))]
                
                var flowRate int
                if rand.Float32() < 0.5 {
                    flowRate = 200 + rand.Intn(100) // Высокая аномалия
                } else {
                    flowRate = rand.Intn(10) // Низкая аномалия
                }

                _, err := dg.pool.Exec(ctx, `
                    INSERT INTO cold_water_meters (id, itp_id, flow_rate, timestamp, created_at)
                    VALUES ($1, $2, $3, $4, NOW())`,
                    uuid.New(), itpID, flowRate, time.Now())
                
                if err != nil {
                    fmt.Printf("Error creating anomaly: %v\n", err)
                } else {
                    fmt.Printf("Anomaly created: ITP %s, flow rate %d\n", itpID, flowRate)
                }
            }
        }
    }
}