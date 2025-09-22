package service

import (
    "context"
    "fmt"
    "math/rand"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/google/uuid"
)

// Простой генератор данных для демонстрации
type DataGenerator struct {
    pool *pgxpool.Pool
}

func NewDataGenerator(pool *pgxpool.Pool) *DataGenerator {
    return &DataGenerator{pool: pool}
}

func (dg *DataGenerator) Start(ctx context.Context) {
    go dg.generateData(ctx)
    fmt.Println("Data generation started")
}

func (dg *DataGenerator) generateData(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second) // Каждые 30 секунд
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            dg.insertDemoData(ctx)
        }
    }
}

func (dg *DataGenerator) insertDemoData(ctx context.Context) {
    // Получаем список зданий
    rows, err := dg.pool.Query(ctx, "SELECT id FROM buildings")
    if err != nil {
        return
    }
    defer rows.Close()

    var buildingIDs []uuid.UUID
    for rows.Next() {
        var id uuid.UUID
        if err := rows.Scan(&id); err != nil {
            continue
        }
        buildingIDs = append(buildingIDs, id)
    }

    if len(buildingIDs) == 0 {
        return
    }

    // Вставляем демо-данные для каждого здания
    for _, buildingID := range buildingIDs {
        // Данные ГВС
        _, err := dg.pool.Exec(ctx, `
            INSERT INTO hot_water_meters (id, building_id, flow_rate_ch1, flow_rate_ch2, timestamp, created_at)
            VALUES ($1, $2, $3, $4, $5, NOW())`,
            uuid.New(), buildingID, 
            20+rand.Intn(30), // flow_rate_ch1: 20-50
            10+rand.Intn(20), // flow_rate_ch2: 10-30
            time.Now())
        
        if err != nil {
            fmt.Printf("Error inserting hot water data: %v\n", err)
        }
    }

    fmt.Println("Demo data inserted at", time.Now().Format("15:04:05"))
}