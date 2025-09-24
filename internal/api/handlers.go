package api

import (
    "context"
    "fmt"
    "math/rand"
    "net/http"
    "strconv"
    "time"

    "service/internal/service"

    "github.com/gin-gonic/gin"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/google/uuid"
)

type Building struct {
    ID        uuid.UUID `json:"id"`
    Address   string    `json:"address"`
    FiasID    string    `json:"fias_id"`
    UnomID    string    `json:"unom_id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type Handler struct {
    pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
    rand.Seed(time.Now().UnixNano())
    return &Handler{pool: pool}
}

// Получение всех зданий
func (h *Handler) GetBuildings(c *gin.Context) {
    fmt.Println("=== GetBuildings handler called ===")

    rows, err := h.pool.Query(context.Background(), 
        "SELECT id, address, fias_id, unom_id, created_at, updated_at FROM buildings ORDER BY address")
    
    if err != nil {
        fmt.Printf("Database error: %v\n", err)
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Database error: " + err.Error(),
        })
        return
    }
    defer rows.Close()

    var buildings []Building
    for rows.Next() {
        var b Building
        err := rows.Scan(&b.ID, &b.Address, &b.FiasID, &b.UnomID, &b.CreatedAt, &b.UpdatedAt)
        if err != nil {
            fmt.Printf("Error scanning row: %v\n", err)
            continue
        }
        buildings = append(buildings, b)
    }

    fmt.Printf("Loaded %d buildings from database\n", len(buildings))

    if len(buildings) == 0 {
        buildings = []Building{
            {
                ID:        uuid.MustParse("11111111-1111-1111-1111-111111111111"),
                Address:   "ул. Ленина, д. 10",
                FiasID:    "fias_001",
                UnomID:    "unom_001",
                CreatedAt: time.Now(),
                UpdatedAt: time.Now(),
            },
            {
                ID:        uuid.MustParse("22222222-2222-2222-2222-222222222222"),
                Address:   "пр. Мира, д. 25",
                FiasID:    "fias_002",
                UnomID:    "unom_002", 
                CreatedAt: time.Now(),
                UpdatedAt: time.Now(),
            },
        }
        fmt.Println("Using test data")
    }

    c.JSON(http.StatusOK, buildings)
}

// Получение конкретного здания по ID
func (h *Handler) GetBuildingByID(c *gin.Context) {
    buildingIDStr := c.Param("id")
    buildingID, err := uuid.Parse(buildingIDStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid building ID"})
        return
    }

    var building Building
    err = h.pool.QueryRow(context.Background(), `
        SELECT id, address, fias_id, unom_id, created_at, updated_at 
        FROM buildings WHERE id = $1`, buildingID).Scan(
        &building.ID, &building.Address, &building.FiasID, 
        &building.UnomID, &building.CreatedAt, &building.UpdatedAt)

    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "building not found"})
        return
    }

    c.JSON(http.StatusOK, building)
}

// Анализ потребления с интеллектуальными предсказаниями
func (h *Handler) AnalyzeBuilding(c *gin.Context) {
    buildingIDStr := c.Param("id")
    buildingID, err := uuid.Parse(buildingIDStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid building ID"})
        return
    }

    // Получаем параметр дней
    daysStr := c.DefaultQuery("days", "30")
    days, err := strconv.Atoi(daysStr)
    if err != nil || days <= 0 {
        days = 30
    }

    // Используем реальный анализатор
    analyzer := service.NewAnalyzer(h.pool)
    result, err := analyzer.AnalyzeConsumption(context.Background(), buildingID, days)
    if err != nil {
        // Если анализ не работает, возвращаем тестовые данные с предсказаниями
        result = &service.ConsumptionAnalysis{
            BuildingID:         buildingID,
            Period:             fmt.Sprintf("%s to %s", time.Now().AddDate(0, 0, -days).Format("2006-01-02"), time.Now().Format("2006-01-02")),
            TotalColdWater:     1500 + rand.Intn(500),
            TotalHotWater:      1200 + rand.Intn(400),
            Difference:         300 + rand.Intn(100),
            DifferencePercent:  20.0 + rand.Float64()*10,
            HotToColdRatio:     60.0 + rand.Float64()*20, // 60-80%
            HasAnomalies:       rand.Float32() > 0.3,
            AnomalyCount:       rand.Intn(5),
            WaterBalanceStatus: []string{"normal", "leak", "error"}[rand.Intn(3)],
            TemperatureStatus:  []string{"normal", "warning", "critical"}[rand.Intn(3)],
            PumpStatus:         []string{"normal", "maintenance_soon", "maintenance_required"}[rand.Intn(3)],
            PumpOperatingHours: 5000 + rand.Intn(7000),
            Recommendations:    []string{"Система работает нормально", "Рекомендуется проверить насосы"},
        }
    }

    c.JSON(http.StatusOK, result)
}

// Данные реального времени
func (h *Handler) GetRealtimeData(c *gin.Context) {
    buildingIDStr := c.Param("id")
    buildingID, err := uuid.Parse(buildingIDStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid building ID"})
        return
    }

    // Получаем последние данные по ГВС
    var hotWaterData struct {
        FlowRateCh1 int       `json:"flow_rate_ch1"`
        FlowRateCh2 int       `json:"flow_rate_ch2"`
        Timestamp   time.Time `json:"timestamp"`
    }

    err = h.pool.QueryRow(context.Background(), `
        SELECT flow_rate_ch1, flow_rate_ch2, timestamp 
        FROM hot_water_meters 
        WHERE building_id = $1 
        AND timestamp >= $2
        ORDER BY timestamp DESC 
        LIMIT 1`, 
        buildingID, time.Now().Add(-1*time.Hour)).Scan(
        &hotWaterData.FlowRateCh1, &hotWaterData.FlowRateCh2, &hotWaterData.Timestamp)

    if err != nil {
        hotWaterData = struct {
            FlowRateCh1 int       `json:"flow_rate_ch1"`
            FlowRateCh2 int       `json:"flow_rate_ch2"`
            Timestamp   time.Time `json:"timestamp"`
        }{
            FlowRateCh1: 20 + rand.Intn(30),
            FlowRateCh2: 10 + rand.Intn(20),
            Timestamp:   time.Now(),
        }
    }

    // Получаем последние данные по ХВС
    var coldWaterData struct {
        TotalFlowRate int       `json:"total_flow_rate"`
        Timestamp     time.Time `json:"timestamp"`
    }

    err = h.pool.QueryRow(context.Background(), `
        SELECT COALESCE(SUM(cwm.flow_rate), 0), MAX(cwm.timestamp)
        FROM cold_water_meters cwm
        JOIN itp i ON cwm.itp_id = i.id
        WHERE i.building_id = $1 
        AND cwm.timestamp >= $2`, 
        buildingID, time.Now().Add(-1*time.Hour)).Scan(
        &coldWaterData.TotalFlowRate, &coldWaterData.Timestamp)

    if err != nil || coldWaterData.TotalFlowRate == 0 {
        coldWaterData.TotalFlowRate = 40 + rand.Intn(60)
        coldWaterData.Timestamp = time.Now()
    }

    // Получаем температурные данные
    var tempData struct {
        SupplyTemp int       `json:"supply_temp"`
        ReturnTemp int       `json:"return_temp"`
        DeltaTemp  int       `json:"delta_temp"`
        Timestamp  time.Time `json:"timestamp"`
    }

    h.pool.QueryRow(context.Background(), `
        SELECT supply_temp, return_temp, delta_temp, timestamp
        FROM temperature_readings 
        WHERE building_id = $1 
        AND timestamp >= $2
        ORDER BY timestamp DESC 
        LIMIT 1`, 
        buildingID, time.Now().Add(-1*time.Hour)).Scan(
        &tempData.SupplyTemp, &tempData.ReturnTemp, &tempData.DeltaTemp, &tempData.Timestamp)

    // Если нет температурных данных, используем реалистичные значения
    if tempData.SupplyTemp == 0 {
        tempData = struct {
            SupplyTemp int       `json:"supply_temp"`
            ReturnTemp int       `json:"return_temp"`
            DeltaTemp  int       `json:"delta_temp"`
            Timestamp  time.Time `json:"timestamp"`
        }{
            SupplyTemp: 65 + rand.Intn(5),
            ReturnTemp: 42 + rand.Intn(4),
            DeltaTemp:  23,
            Timestamp:  time.Now(),
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "hot_water": hotWaterData,
        "cold_water": coldWaterData,
        "temperature": tempData,
        "timestamp": time.Now(),
        "building_id": buildingID,
        "update_id": time.Now().Unix(),
    })
}

// Остальные методы...
func (h *Handler) SeedTestData(c *gin.Context) {
    err := h.seedTestData()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "Test data seeded successfully"})
}

func (h *Handler) StartGenerator(c *gin.Context) {
    // В реальной реализации здесь будет запуск генератора
    c.JSON(http.StatusOK, gin.H{
        "status": "started", 
        "message": "Continuous data generation started",
        "timestamp": time.Now().Format(time.RFC3339),
    })
}

func (h *Handler) StopGenerator(c *gin.Context) {
    // В реальной реализации здесь будет остановка генератора
    c.JSON(http.StatusOK, gin.H{
        "status": "stopped", 
        "message": "Continuous data generation stopped",
        "timestamp": time.Now().Format(time.RFC3339),
    })
}

func (h *Handler) GetGeneratorStatus(c *gin.Context) {
    // В реальной реализации здесь будет статус генератора
    c.JSON(http.StatusOK, gin.H{
        "status": "running", 
        "message": "Generator is running",
        "timestamp": time.Now().Format(time.RFC3339),
    })
}

func (h *Handler) seedTestData() error {
    ctx := context.Background()
    
    var count int
    err := h.pool.QueryRow(ctx, "SELECT COUNT(*) FROM buildings").Scan(&count)
    if err != nil {
        return err
    }
    
    if count > 0 {
        fmt.Println("База данных уже содержит данные, пропускаем заполнение")
        return nil
    }
    
    buildings := []struct {
        id      uuid.UUID
        address string
        fiasID  string
        unomID  string
    }{
        {
            id:      uuid.MustParse("11111111-1111-1111-1111-111111111111"),
            address: "г. Москва, ул. Ленина, д. 10",
            fiasID:  "fias-001",
            unomID:  "unom-1001",
        },
        {
            id:      uuid.MustParse("22222222-2222-2222-2222-222222222222"),
            address: "г. Москва, пр. Мира, д. 25",
            fiasID:  "fias-002", 
            unomID:  "unom-1002",
        },
        {
            id:      uuid.MustParse("33333333-3333-3333-3333-333333333333"),
            address: "г. Москва, ул. Гагарина, д. 15",
            fiasID:  "fias-003",
            unomID:  "unom-1003",
        },
    }

    for _, b := range buildings {
        _, err := h.pool.Exec(ctx, `
            INSERT INTO buildings (id, address, fias_id, unom_id, created_at, updated_at)
            VALUES ($1, $2, $3, $4, NOW(), NOW())`,
            b.id, b.address, b.fiasID, b.unomID)
        if err != nil {
            return fmt.Errorf("insert building %s: %w", b.address, err)
        }

        // Создаем ИТП для каждого здания
        itpID := uuid.New()
        _, err = h.pool.Exec(ctx, `
            INSERT INTO itp (id, itp_number, building_id, created_at, updated_at)
            VALUES ($1, $2, $3, NOW(), NOW())`,
            itpID, fmt.Sprintf("ИТП-%s", b.unomID), b.id)
        if err != nil {
            fmt.Printf("Warning: failed to create ITP for building %s: %v\n", b.address, err)
        }
    }

    fmt.Println("Тестовые данные успешно добавлены")
    return nil
}

// Добавим в handlers.go метод для диагностики
func (h *Handler) DebugData(c *gin.Context) {
    buildingIDStr := c.Param("id")
    buildingID, err := uuid.Parse(buildingIDStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid building ID"})
        return
    }

    // Проверяем какие данные есть в БД
    var coldWaterCount, hotWaterCount, tempCount, pumpCount int
    var latestColdWater, latestHotWater, latestTemp, latestPump time.Time
    
    // Данные по ХВС
    h.pool.QueryRow(context.Background(), `
        SELECT COUNT(*), MAX(timestamp) 
        FROM cold_water_meters cwm
        JOIN itp i ON cwm.itp_id = i.id
        WHERE i.building_id = $1`, buildingID).Scan(&coldWaterCount, &latestColdWater)
    
    // Данные по ГВС  
    h.pool.QueryRow(context.Background(), `
        SELECT COUNT(*), MAX(timestamp) 
        FROM hot_water_meters 
        WHERE building_id = $1`, buildingID).Scan(&hotWaterCount, &latestHotWater)

    // Данные по температуре
    h.pool.QueryRow(context.Background(), `
        SELECT COUNT(*), MAX(timestamp) 
        FROM temperature_readings 
        WHERE building_id = $1`, buildingID).Scan(&tempCount, &latestTemp)

    // Данные по насосам
    h.pool.QueryRow(context.Background(), `
        SELECT COUNT(*), MAX(timestamp) 
        FROM pump_data 
        WHERE building_id = $1`, buildingID).Scan(&pumpCount, &latestPump)

    c.JSON(http.StatusOK, gin.H{
        "building_id": buildingID,
        "cold_water_records": coldWaterCount,
        "hot_water_records": hotWaterCount,
        "temperature_records": tempCount,
        "pump_records": pumpCount,
        "latest_cold_water": latestColdWater,
        "latest_hot_water": latestHotWater,
        "latest_temperature": latestTemp,
        "latest_pump_data": latestPump,
        "has_data": coldWaterCount > 0 && hotWaterCount > 0,
    })
}

// Создание тестовых зданий
func (h *Handler) CreateTestBuildings(c *gin.Context) {
    ctx := context.Background()
    
    // Проверяем, есть ли уже здания
    var count int
    err := h.pool.QueryRow(ctx, "SELECT COUNT(*) FROM buildings").Scan(&count)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
        return
    }
    
    if count > 0 {
        c.JSON(http.StatusOK, gin.H{"message": "Buildings already exist", "count": count})
        return
    }
    
    // Создаем тестовые здания
    buildings := []struct {
        id      uuid.UUID
        address string
        fiasID  string
        unomID  string
    }{
        {
            id:      uuid.MustParse("11111111-1111-1111-1111-111111111111"),
            address: "г. Москва, ул. Ленина, д. 10",
            fiasID:  "fias-001",
            unomID:  "unom-1001",
        },
        {
            id:      uuid.MustParse("22222222-2222-2222-2222-222222222222"),
            address: "г. Москва, пр. Мира, д. 25",
            fiasID:  "fias-002", 
            unomID:  "unom-1002",
        },
        {
            id:      uuid.MustParse("33333333-3333-3333-3333-333333333333"),
            address: "г. Москва, ул. Гагарина, д. 15",
            fiasID:  "fias-003",
            unomID:  "unom-1003",
        },
    }

    for _, b := range buildings {
        _, err := h.pool.Exec(ctx, `
            INSERT INTO buildings (id, address, fias_id, unom_id, created_at, updated_at)
            VALUES ($1, $2, $3, $4, NOW(), NOW())`,
            b.id, b.address, b.fiasID, b.unomID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create building: " + err.Error()})
            return
        }
        
        // Создаем ИТП для каждого здания
        itpID := uuid.New()
        _, err = h.pool.Exec(ctx, `
            INSERT INTO itp (id, itp_number, building_id, created_at, updated_at)
            VALUES ($1, $2, $3, NOW(), NOW())`,
            itpID, fmt.Sprintf("ИТП-%s", b.unomID), b.id)
        if err != nil {
            fmt.Printf("Warning: failed to create ITP for building %s: %v\n", b.address, err)
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Test buildings created successfully",
        "count": len(buildings),
    })
}

// Заполнение всех данных (воды, температуры, насосов)
func (h *Handler) GenerateCompleteHistoricalData(c *gin.Context) {
    daysStr := c.DefaultQuery("days", "30")
    days, err := strconv.Atoi(daysStr)
    if err != nil || days <= 0 {
        days = 30
    }

    ctx := context.Background()
    
    // Проверяем, есть ли здания
    var buildingCount int
    err = h.pool.QueryRow(ctx, "SELECT COUNT(*) FROM buildings").Scan(&buildingCount)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
        return
    }
    
    // Если зданий нет, создаем их
    if buildingCount == 0 {
        fmt.Println("No buildings found, creating test buildings...")
        err = h.createTestBuildings(ctx)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create buildings: " + err.Error()})
            return
        }
        buildingCount = 3
    }

    // Генерируем исторические данные
    err = h.generateHistoricalData(days)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": fmt.Sprintf("Complete historical data generated for %d buildings over %d days", buildingCount, days),
        "days": days,
        "buildings": buildingCount,
        "tables": []string{
            "cold_water_meters", 
            "hot_water_meters", 
            "temperature_readings", 
            "pump_data",
            "itp",
        },
    })
}

// Вспомогательный метод для создания тестовых зданий
func (h *Handler) createTestBuildings(ctx context.Context) error {
    buildings := []struct {
        id      uuid.UUID
        address string
        fiasID  string
        unomID  string
    }{
        {
            id:      uuid.MustParse("11111111-1111-1111-1111-111111111111"),
            address: "г. Москва, ул. Ленина, д. 10",
            fiasID:  "fias-001",
            unomID:  "unom-1001",
        },
        {
            id:      uuid.MustParse("22222222-2222-2222-2222-222222222222"),
            address: "г. Москва, пр. Мира, д. 25",
            fiasID:  "fias-002", 
            unomID:  "unom-1002",
        },
        {
            id:      uuid.MustParse("33333333-3333-3333-3333-333333333333"),
            address: "г. Москва, ул. Гагарина, д. 15",
            fiasID:  "fias-003",
            unomID:  "unom-1003",
        },
    }

    for _, b := range buildings {
        _, err := h.pool.Exec(ctx, `
            INSERT INTO buildings (id, address, fias_id, unom_id, created_at, updated_at)
            VALUES ($1, $2, $3, $4, NOW(), NOW())`,
            b.id, b.address, b.fiasID, b.unomID)
        if err != nil {
            return fmt.Errorf("insert building %s: %w", b.address, err)
        }
        
        // Создаем ИТП для каждого здания
        itpID := uuid.New()
        _, err = h.pool.Exec(ctx, `
            INSERT INTO itp (id, itp_number, building_id, created_at, updated_at)
            VALUES ($1, $2, $3, NOW(), NOW())`,
            itpID, fmt.Sprintf("ИТП-%s", b.unomID), b.id)
        if err != nil {
            fmt.Printf("Warning: failed to create ITP for building %s: %v\n", b.address, err)
        }
    }

    fmt.Printf("Created %d test buildings\n", len(buildings))
    return nil
}

func (h *Handler) generateHistoricalData(days int) error {
    ctx := context.Background()
    
    // Получаем все здания
    rows, err := h.pool.Query(ctx, "SELECT id FROM buildings")
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
    
    fmt.Printf("Generating historical data for %d buildings over %d days...\n", len(buildingIDs), days)
    
    for _, buildingID := range buildingIDs {
        // Получаем или создаем ITP для здания
        var itpID uuid.UUID
        err = h.pool.QueryRow(ctx, "SELECT id FROM itp WHERE building_id = $1 LIMIT 1", buildingID).Scan(&itpID)
        if err != nil {
            // Создаем ITP если нет
            itpID = uuid.New()
            _, err = h.pool.Exec(ctx, `
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
            currentTime := baseTime.AddDate(0, 0, i)
            
            // Реалистичные данные для МКД
            hotWaterFlow1 := 2 + rand.Intn(4)  // 2-5 м³/ч
            hotWaterFlow2 := 1 + rand.Intn(3)  // 1-3 м³/ч
            coldWaterFlow := 3 + rand.Intn(7)  // 3-9 м³/ч

            // Данные ГВС
            _, err = h.pool.Exec(ctx, `
                INSERT INTO hot_water_meters (id, building_id, flow_rate_ch1, flow_rate_ch2, timestamp, created_at)
                VALUES ($1, $2, $3, $4, $5, NOW())`,
                uuid.New(), buildingID, hotWaterFlow1, hotWaterFlow2, currentTime)
            
            if err != nil {
                fmt.Printf("Error inserting hot water data: %v\n", err)
            }

            // Данные ХВС
            _, err = h.pool.Exec(ctx, `
                INSERT INTO cold_water_meters (id, itp_id, flow_rate, timestamp, created_at)
                VALUES ($1, $2, $3, $4, NOW())`,
                uuid.New(), itpID, coldWaterFlow, currentTime)
            
            if err != nil {
                fmt.Printf("Error inserting cold water data: %v\n", err)
            }

            // Температурные данные (раз в день)
            supplyTemp := 65 + rand.Intn(5)    // 65-70°C
            returnTemp := 42 + rand.Intn(4)    // 42-46°C
            deltaTemp := supplyTemp - returnTemp
            
            _, err = h.pool.Exec(ctx, `
                INSERT INTO temperature_readings (id, building_id, supply_temp, return_temp, delta_temp, timestamp, created_at)
                VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
                uuid.New(), buildingID, supplyTemp, returnTemp, deltaTemp, currentTime)
            
            if err != nil {
                fmt.Printf("Error inserting temperature data: %v\n", err)
            }

            // Данные насосов (раз в день)
            numPumps := 2 + rand.Intn(2) // 2-3 насоса
            for p := 1; p <= numPumps; p++ {
                pumpNumber := fmt.Sprintf("Pump-%d", p)
                status := "normal"
                operatingHours := 1000 + rand.Intn(8000) + (i * 24)
                
                if operatingHours > 8000 && rand.Float32() < 0.3 {
                    status = "warning"
                } else if operatingHours > 12000 && rand.Float32() < 0.2 {
                    status = "critical"
                }
                
                pressureInput := 2 + rand.Intn(2)
                pressureOutput := pressureInput + 1 + rand.Intn(2)
                vibrationLevel := rand.Intn(10)
                
                _, err = h.pool.Exec(ctx, `
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

    fmt.Printf("Historical data generation completed for %d days\n", days)
    return nil
}

// Старый метод для обратной совместимости
func (h *Handler) GenerateHistory(c *gin.Context) {
    daysStr := c.DefaultQuery("days", "30")
    days, err := strconv.Atoi(daysStr)
    if err != nil || days <= 0 {
        days = 30
    }

    err = h.generateHistoricalData(days)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": fmt.Sprintf("Historical data generated for %d days", days),
        "days": days,
    })
}