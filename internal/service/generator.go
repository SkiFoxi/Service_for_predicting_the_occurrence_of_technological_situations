package service

import (
    "context"
    "fmt"
    "math/rand"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/google/uuid"
)

// Расширенный генератор данных для реального времени
type DataGenerator struct {
    pool      *pgxpool.Pool
    isRunning bool
    ctx       context.Context
    cancel    context.CancelFunc
}

func NewDataGenerator(pool *pgxpool.Pool) *DataGenerator {
    return &DataGenerator{pool: pool}
}

// Запуск непрерывной генерации данных
func (dg *DataGenerator) StartContinuousGeneration(ctx context.Context) {
    if dg.isRunning {
        fmt.Println("Generator is already running")
        return
    }

    dg.ctx, dg.cancel = context.WithCancel(ctx)
    dg.isRunning = true

    fmt.Println("🚀 Starting continuous data generation...")

    // Запускаем различные тикеры для разных типов данных
    go dg.startWaterDataGeneration()
    go dg.startTemperatureDataGeneration()
    go dg.startPumpDataGeneration()
    go dg.startRealtimeUpdates()

    fmt.Println("✅ Continuous data generation started")
}

// Остановка генерации
func (dg *DataGenerator) Stop() {
    if dg.isRunning && dg.cancel != nil {
        dg.cancel()
        dg.isRunning = false
        fmt.Println("🛑 Data generation stopped")
    }
}

// Получение статуса генератора
func (dg *DataGenerator) IsRunning() bool {
    return dg.isRunning
}


// Генерация водных данных (каждые 30 секунд)
func (dg *DataGenerator) startWaterDataGeneration() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-dg.ctx.Done():
            return
        case <-ticker.C:
            dg.generateWaterData()
        }
    }
}

// Генерация температурных данных (каждые 2 минуты)
func (dg *DataGenerator) startTemperatureDataGeneration() {
    ticker := time.NewTicker(2 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-dg.ctx.Done():
            return
        case <-ticker.C:
            dg.generateTemperatureDataForAllBuildings()
        }
    }
}

// Генерация данных насосов (каждые 5 минут)
func (dg *DataGenerator) startPumpDataGeneration() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-dg.ctx.Done():
            return
        case <-ticker.C:
            dg.generatePumpDataForAllBuildings()
        }
    }
}

// Уведомления о новых данных (для веб-сокетов)
func (dg *DataGenerator) startRealtimeUpdates() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-dg.ctx.Done():
            return
        case <-ticker.C:
            dg.broadcastDataUpdate()
        }
    }
}

// Генерация водных данных для всех зданий
func (dg *DataGenerator) generateWaterData() {
    buildings, err := dg.getBuildings()
    if err != nil {
        fmt.Printf("Error getting buildings: %v\n", err)
        return
    }

    currentTime := time.Now()
    
    for _, building := range buildings {
        // Реалистичные данные с небольшими случайными колебаниями
        baseHotWater1 := 2.5 + rand.Float64()*2.0  // 2.5-4.5 м³/ч
        baseHotWater2 := 1.5 + rand.Float64()*1.5  // 1.5-3.0 м³/ч
        baseColdWater := baseHotWater1 + baseHotWater2 + 1.0 + rand.Float64()*2.0 // ХВС > ГВС

        // Добавляем суточные колебания (утром/вечером больше потребление)
        hour := currentTime.Hour()
        var dailyMultiplier float64
        switch {
        case hour >= 7 && hour <= 10: // Утро
            dailyMultiplier = 1.3
        case hour >= 18 && hour <= 22: // Вечер
            dailyMultiplier = 1.4
        case hour >= 23 || hour <= 6: // Ночь
            dailyMultiplier = 0.7
        default: // День
            dailyMultiplier = 1.1
        }

        hotWater1 := int(baseHotWater1 * dailyMultiplier)
        hotWater2 := int(baseHotWater2 * dailyMultiplier)
        coldWater := int(baseColdWater * dailyMultiplier)

        // Данные ГВС
        _, err := dg.pool.Exec(dg.ctx, `
            INSERT INTO hot_water_meters (id, building_id, flow_rate_ch1, flow_rate_ch2, timestamp, created_at)
            VALUES ($1, $2, $3, $4, $5, NOW())`,
            uuid.New(), building.ID, hotWater1, hotWater2, currentTime)
        
        if err != nil {
            fmt.Printf("Error inserting hot water data: %v\n", err)
            continue
        }

        // Данные ХВС
        itpID, err := dg.getITPForBuilding(building.ID)
        if err == nil {
            _, err = dg.pool.Exec(dg.ctx, `
                INSERT INTO cold_water_meters (id, itp_id, flow_rate, timestamp, created_at)
                VALUES ($1, $2, $3, $4, NOW())`,
                uuid.New(), itpID, coldWater, currentTime)
            
            if err != nil {
                fmt.Printf("Error inserting cold water data: %v\n", err)
            }
        }
    }

    fmt.Printf("💧 Water data generated for %d buildings at %s\n", len(buildings), currentTime.Format("15:04:05"))
}

// Генерация температурных данных для всех зданий
func (dg *DataGenerator) generateTemperatureDataForAllBuildings() {
    buildings, err := dg.getBuildings()
    if err != nil {
        fmt.Printf("Error getting buildings: %v\n", err)
        return
    }

    currentTime := time.Now()
    
    for _, building := range buildings {
        // Реалистичные температурные данные с сезонными колебаниями
        month := currentTime.Month()
        var seasonalAdjustment int
        
        switch month {
        case time.December, time.January, time.February: // Зима
            seasonalAdjustment = 5
        case time.June, time.July, time.August: // Лето
            seasonalAdjustment = -3
        default: // Весна/осень
            seasonalAdjustment = 0
        }

        supplyTemp := 65 + seasonalAdjustment + rand.Intn(5)    // 65-70°C ± сезонная корректировка
        returnTemp := 42 + seasonalAdjustment/2 + rand.Intn(4)  // 42-46°C
        deltaTemp := supplyTemp - returnTemp

        _, err := dg.pool.Exec(dg.ctx, `
            INSERT INTO temperature_readings (id, building_id, supply_temp, return_temp, delta_temp, timestamp, created_at)
            VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
            uuid.New(), building.ID, supplyTemp, returnTemp, deltaTemp, currentTime)
        
        if err != nil {
            fmt.Printf("Error inserting temperature data: %v\n", err)
        }
    }

    fmt.Printf("🌡️ Temperature data generated for %d buildings at %s\n", len(buildings), currentTime.Format("15:04:05"))
}

// Генерация данных насосов для всех зданий
func (dg *DataGenerator) generatePumpDataForAllBuildings() {
    buildings, err := dg.getBuildings()
    if err != nil {
        fmt.Printf("Error getting buildings: %v\n", err)
        return
    }

    currentTime := time.Now()
    
    for _, building := range buildings {
        // Генерируем данные для 2-3 насосов на здание
        numPumps := 2 + rand.Intn(2)
        
        for i := 1; i <= numPumps; i++ {
            pumpNumber := fmt.Sprintf("Pump-%d", i)
            
            // Наработка увеличивается с каждым обновлением
            baseHours := 5000 + rand.Intn(3000)
            additionalHours := int(time.Since(building.CreatedAt).Hours()) / 24
            operatingHours := baseHours + additionalHours

            // Статус зависит от наработки
            status := "normal"
            if operatingHours > 10000 && rand.Float32() < 0.4 {
                status = "warning"
            } else if operatingHours > 15000 && rand.Float32() < 0.3 {
                status = "critical"
            }

            pressureInput := 2 + rand.Intn(2)
            pressureOutput := pressureInput + 1 + rand.Intn(2)
            vibrationLevel := rand.Intn(8) // 0-7

            _, err := dg.pool.Exec(dg.ctx, `
                INSERT INTO pump_data (id, building_id, pump_number, status, operating_hours, 
                                     pressure_input, pressure_output, vibration_level, timestamp, created_at)
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())`,
                uuid.New(), building.ID, pumpNumber, status, operatingHours, 
                pressureInput, pressureOutput, vibrationLevel, currentTime)
            
            if err != nil {
                fmt.Printf("Error inserting pump data: %v\n", err)
            }
        }
    }

    fmt.Printf("⚙️ Pump data generated for %d buildings at %s\n", len(buildings), currentTime.Format("15:04:05"))
}

// Уведомление о новых данных (заглушка для веб-сокетов)
func (dg *DataGenerator) broadcastDataUpdate() {
    // Здесь будет логика для веб-сокетов
    // Пока просто логируем
    fmt.Printf("📡 Data update broadcast at %s\n", time.Now().Format("15:04:05"))
}

// Вспомогательные методы
func (dg *DataGenerator) getBuildings() ([]Building, error) {
    rows, err := dg.pool.Query(dg.ctx, "SELECT id, address, created_at FROM buildings")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var buildings []Building
    for rows.Next() {
        var b Building
        err := rows.Scan(&b.ID, &b.Address, &b.CreatedAt)
        if err != nil {
            continue
        }
        buildings = append(buildings, b)
    }

    return buildings, nil
}

func (dg *DataGenerator) getITPForBuilding(buildingID uuid.UUID) (uuid.UUID, error) {
    var itpID uuid.UUID
    err := dg.pool.QueryRow(dg.ctx, 
        "SELECT id FROM itp WHERE building_id = $1 LIMIT 1", buildingID).Scan(&itpID)
    return itpID, err
}

// Структура для зданий
type Building struct {
    ID        uuid.UUID
    Address   string
    CreatedAt time.Time
}

// Старые методы для обратной совместимости
func (dg *DataGenerator) Start(ctx context.Context) {
    dg.StartContinuousGeneration(ctx)
}

func (dg *DataGenerator) insertDemoData(ctx context.Context) {
    dg.generateWaterData()
}

// generator.go - добавьте эти методы в конец файла

// Генерация полных исторических данных
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
                    fmt.Printf("Error inserting hot water data: %v\n", err)
                }

                // Данные ХВС
                _, err = dg.pool.Exec(ctx, `
                    INSERT INTO cold_water_meters (id, itp_id, flow_rate, timestamp, created_at)
                    VALUES ($1, $2, $3, $4, NOW())`,
                    uuid.New(), itpID, coldWaterFlow, currentTime)
                
                if err != nil {
                    fmt.Printf("Error inserting cold water data: %v\n", err)
                }
            }

            // Температурные данные (раз в день)
            supplyTemp := 65 + rand.Intn(5)
            returnTemp := 42 + rand.Intn(4)
            deltaTemp := supplyTemp - returnTemp
            
            _, err = dg.pool.Exec(ctx, `
                INSERT INTO temperature_readings (id, building_id, supply_temp, return_temp, delta_temp, timestamp, created_at)
                VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
                uuid.New(), buildingID, supplyTemp, returnTemp, deltaTemp, currentDay)
            
            if err != nil {
                fmt.Printf("Error inserting temperature data: %v\n", err)
            }

            // Данные насосов (раз в день)
            numPumps := 2 + rand.Intn(2)
            for p := 1; p <= numPumps; p++ {
                pumpNumber := fmt.Sprintf("Pump-%d", p)
                status := "normal"
                operatingHours := 1000 + rand.Intn(8000) + (i * 24)
                
                if operatingHours > 8000 && rand.Float32() < 0.3 {
                    status = "warning"
                }
                
                pressureInput := 2 + rand.Intn(2)
                pressureOutput := pressureInput + 1 + rand.Intn(2)
                vibrationLevel := rand.Intn(10)
                
                _, err = dg.pool.Exec(ctx, `
                    INSERT INTO pump_data (id, building_id, pump_number, status, operating_hours, 
                                         pressure_input, pressure_output, vibration_level, timestamp, created_at)
                    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())`,
                    uuid.New(), buildingID, pumpNumber, status, operatingHours, 
                    pressureInput, pressureOutput, vibrationLevel, currentDay)
                
                if err != nil {
                    fmt.Printf("Error inserting pump data: %v\n", err)
                }
            }
        }
    }

    fmt.Printf("Complete historical data generation completed for %d days\n", days)
    return nil
}

// Старый метод для обратной совместимости
func (dg *DataGenerator) GenerateHistoricalData(ctx context.Context, days int) error {
    return dg.GenerateCompleteHistoricalData(ctx, days)
}
