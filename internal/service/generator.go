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
        fmt.Printf("Error getting buildings: %v\n", err)
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
        fmt.Println("No buildings found for data generation")
        return
    }

    // Вставляем демо-данные для каждого здания
    for _, buildingID := range buildingIDs {
        // РЕАЛИСТИЧНЫЕ данные для МКД:
        
        // Данные ГВС - реалистичные значения
        hotWaterFlow1 := 2 + rand.Intn(4)  // 2-5 м³/ч - канал 1
        hotWaterFlow2 := 1 + rand.Intn(3)  // 1-3 м³/ч - канал 2

        _, err := dg.pool.Exec(ctx, `
            INSERT INTO hot_water_meters (id, building_id, flow_rate_ch1, flow_rate_ch2, timestamp, created_at)
            VALUES ($1, $2, $3, $4, $5, NOW())`,
            uuid.New(), buildingID, hotWaterFlow1, hotWaterFlow2, time.Now())
        
        if err != nil {
            fmt.Printf("Error inserting hot water data: %v\n", err)
        }

        // Данные ХВС - реалистичные значения (должны быть больше ГВС)
        // Сначала получим ITP для этого здания
        var itpID uuid.UUID
        err = dg.pool.QueryRow(ctx, "SELECT id FROM itp WHERE building_id = $1 LIMIT 1", buildingID).Scan(&itpID)
        if err == nil {
            coldWaterFlow := 3 + rand.Intn(7) // 3-9 м³/ч - ХВС должно быть больше ГВС
            
            _, err = dg.pool.Exec(ctx, `
                INSERT INTO cold_water_meters (id, itp_id, flow_rate, timestamp, created_at)
                VALUES ($1, $2, $3, $4, NOW())`,
                uuid.New(), itpID, coldWaterFlow, time.Now())
            
            if err != nil {
                fmt.Printf("Error inserting cold water data: %v\n", err)
            }
        }

        // Температурные данные (генерируем реже - раз в 10 минут)
        if time.Now().Minute()%10 == 0 {
            if err := dg.generateTemperatureData(ctx, buildingID); err != nil {
                fmt.Printf("Error generating temperature data: %v\n", err)
            }
        }

        // Данные насосов (генерируем реже - раз в 30 минут)
        if time.Now().Minute()%30 == 0 {
            if err := dg.generatePumpData(ctx, buildingID); err != nil {
                fmt.Printf("Error generating pump data: %v\n", err)
            }
        }
    }

    fmt.Printf("Demo data inserted for %d buildings at %s\n", len(buildingIDs), time.Now().Format("15:04:05"))
}

// Генерация температурных данных
func (dg *DataGenerator) generateTemperatureData(ctx context.Context, buildingID uuid.UUID) error {
    // Реалистичные температурные данные для ГВС
    supplyTemp := 60 + rand.Intn(10)    // 60-70°C - подача
    returnTemp := 40 + rand.Intn(10)    // 40-50°C - возврат
    deltaTemp := supplyTemp - returnTemp // разница 15-25°C
    
    _, err := dg.pool.Exec(ctx, `
        INSERT INTO temperature_readings (id, building_id, supply_temp, return_temp, delta_temp, timestamp, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
        uuid.New(), buildingID, supplyTemp, returnTemp, deltaTemp, time.Now())
    
    if err != nil {
        return fmt.Errorf("insert temperature data: %w", err)
    }
    return nil
}

// Генерация данных насосов
func (dg *DataGenerator) generatePumpData(ctx context.Context, buildingID uuid.UUID) error {
    // Генерируем данные для 2-4 насосов
    numPumps := 2 + rand.Intn(3)
    
    for i := 1; i <= numPumps; i++ {
        pumpNumber := fmt.Sprintf("Pump-%d", i)
        status := "normal"
        operatingHours := 1000 + rand.Intn(8000)
        
        // Случайно меняем статус для демонстрации
        if rand.Float32() < 0.1 { // 10% chance for warning
            status = "warning"
        } else if rand.Float32() < 0.05 { // 5% chance for critical
            status = "critical"
        }
        
        pressureInput := 2 + rand.Intn(2)    // 2-4 бар
        pressureOutput := pressureInput + 1 + rand.Intn(2) // +1-3 бар
        vibrationLevel := rand.Intn(10)      // 0-9
        
        _, err := dg.pool.Exec(ctx, `
            INSERT INTO pump_data (id, building_id, pump_number, status, operating_hours, 
                                 pressure_input, pressure_output, vibration_level, timestamp, created_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())`,
            uuid.New(), buildingID, pumpNumber, status, operatingHours, 
            pressureInput, pressureOutput, vibrationLevel, time.Now())
        
        if err != nil {
            return fmt.Errorf("insert pump data: %w", err)
        }
    }
    return nil
}

// Новый метод для заполнения историческими данными всех таблиц
func (dg *DataGenerator) GenerateCompleteHistoricalData(ctx context.Context, days int) error {
    // Получаем список зданий
    rows, err := dg.pool.Query(ctx, "SELECT id FROM buildings")
    if err != nil {
        return err
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
        return fmt.Errorf("no buildings found")
    }

    baseTime := time.Now().AddDate(0, 0, -days)
    
    fmt.Printf("Generating complete historical data for %d buildings over %d days...\n", len(buildingIDs), days)
    
    for _, buildingID := range buildingIDs {
        // Получаем ITP для здания
        var itpID uuid.UUID
        err = dg.pool.QueryRow(ctx, "SELECT id FROM itp WHERE building_id = $1 LIMIT 1", buildingID).Scan(&itpID)
        if err != nil {
            // Создаем ITP если нет
            itpID = uuid.New()
            _, err = dg.pool.Exec(ctx, `
                INSERT INTO itp (id, itp_number, building_id, created_at, updated_at)
                VALUES ($1, $2, $3, NOW(), NOW())`,
                itpID, fmt.Sprintf("ИТП-%s", buildingID.String()[:8]), buildingID)
            if err != nil {
                fmt.Printf("Error creating ITP: %v\n", err)
                continue
            }
        }

        // Генерируем данные за каждый день
        for i := 0; i < days; i++ {
            currentDay := baseTime.AddDate(0, 0, i)
            
            // Генерируем несколько записей в день (каждый час)
            for hour := 0; hour < 24; hour++ {
                currentTime := currentDay.Add(time.Duration(hour) * time.Hour)

                // Водные данные
                hotWaterFlow1 := 2 + rand.Intn(4)
                hotWaterFlow2 := 1 + rand.Intn(3)
                coldWaterFlow := 3 + rand.Intn(7)

                // Данные ГВС
                _, err = dg.pool.Exec(ctx, `
                    INSERT INTO hot_water_meters (id, building_id, flow_rate_ch1, flow_rate_ch2, timestamp, created_at)
                    VALUES ($1, $2, $3, $4, $5, NOW())`,
                    uuid.New(), buildingID, hotWaterFlow1, hotWaterFlow2, currentTime)
                
                if err != nil {
                    fmt.Printf("Error inserting historical hot water data: %v\n", err)
                }

                // Данные ХВС
                _, err = dg.pool.Exec(ctx, `
                    INSERT INTO cold_water_meters (id, itp_id, flow_rate, timestamp, created_at)
                    VALUES ($1, $2, $3, $4, NOW())`,
                    uuid.New(), itpID, coldWaterFlow, currentTime)
                
                if err != nil {
                    fmt.Printf("Error inserting historical cold water data: %v\n", err)
                }

                // Температурные данные (раз в 4 часа)
                if hour%4 == 0 {
                    supplyTemp := 60 + rand.Intn(10)
                    returnTemp := 40 + rand.Intn(10)
                    deltaTemp := supplyTemp - returnTemp
                    
                    _, err = dg.pool.Exec(ctx, `
                        INSERT INTO temperature_readings (id, building_id, supply_temp, return_temp, delta_temp, timestamp, created_at)
                        VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
                        uuid.New(), buildingID, supplyTemp, returnTemp, deltaTemp, currentTime)
                    
                    if err != nil {
                        fmt.Printf("Error inserting temperature data: %v\n", err)
                    }
                }

                // Данные насосов (раз в день)
                if hour == 12 { // В полдень каждого дня
                    numPumps := 2 + rand.Intn(3)
                    for p := 1; p <= numPumps; p++ {
                        pumpNumber := fmt.Sprintf("Pump-%d", p)
                        status := "normal"
                        operatingHours := 1000 + rand.Intn(8000) + (i * 24) // увеличиваем наработку с каждым днем
                        
                        if operatingHours > 8000 && rand.Float32() < 0.3 {
                            status = "warning"
                        } else if operatingHours > 12000 && rand.Float32() < 0.2 {
                            status = "critical"
                        }
                        
                        pressureInput := 2 + rand.Intn(2)
                        pressureOutput := pressureInput + 1 + rand.Intn(2)
                        vibrationLevel := rand.Intn(10)
                        
                        _, err = dg.pool.Exec(ctx, `
                            INSERT INTO pump_data (id, building_id, pump_number, status, operating_hours, 
                                                 pressure_input, pressure_output, vibration_level, timestamp, created_at)
                            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())`,
                            uuid.New(), buildingID, pumpNumber, status, operatingHours, 
                            pressureInput, pressureOutput, vibrationLevel, currentTime)
                        
                        if err != nil {
                            fmt.Printf("Error inserting pump data: %v\n", err)
                        }
                    }
                }
            }
        }
    }

    fmt.Printf("Complete historical data generation completed for %d days\n", days)
    return nil
}

// Старый метод для обратной совместимости (можно удалить если не используется)
func (dg *DataGenerator) GenerateHistoricalData(ctx context.Context, days int) error {
    // Просто вызываем новый метод
    return dg.GenerateCompleteHistoricalData(ctx, days)
}