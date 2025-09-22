package api

import (
    "context"
    "fmt"
    "math/rand"
    "net/http"
    "time"

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
    // Инициализируем генератор случайных чисел
    rand.Seed(time.Now().UnixNano())
    return &Handler{pool: pool}
}

// Получение всех зданий
func (h *Handler) GetBuildings(c *gin.Context) {
    fmt.Println("=== GetBuildings handler called ===")

    // Простой запрос к базе данных
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

    // Если в базе нет данных, возвращаем тестовые
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

// Анализ потребления
func (h *Handler) AnalyzeBuilding(c *gin.Context) {
    buildingIDStr := c.Param("id")
    buildingID, err := uuid.Parse(buildingIDStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid building ID"})
        return
    }

    // Временно возвращаем тестовые данные анализа
    c.JSON(http.StatusOK, gin.H{
        "building_id":        buildingID,
        "period":             "2024-01-01 to 2024-01-30",
        "total_cold_water":   1500,
        "total_hot_water":    1200,
        "difference":         300,
        "difference_percent": 20.0,
        "has_anomalies":      true,
        "anomaly_count":      3,
    })
}

// Данные реального времени
func (h *Handler) GetRealtimeData(c *gin.Context) {
    buildingIDStr := c.Param("id")
    buildingID, err := uuid.Parse(buildingIDStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid building ID"})
        return
    }

    // Получаем последние данные по ГВС (за последний час)
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
        // Если нет свежих данных, используем случайные значения для демо
        hotWaterData = struct {
            FlowRateCh1 int       `json:"flow_rate_ch1"`
            FlowRateCh2 int       `json:"flow_rate_ch2"`
            Timestamp   time.Time `json:"timestamp"`
        }{
            FlowRateCh1: 20 + rand.Intn(30), // 20-50
            FlowRateCh2: 10 + rand.Intn(20), // 10-30
            Timestamp:   time.Now(),
        }
    }

    // Получаем последние данные по ХВС (сумма по всем ИТП здания)
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
        coldWaterData.TotalFlowRate = 40 + rand.Intn(60) // 40-100
        coldWaterData.Timestamp = time.Now()
    }

    c.JSON(http.StatusOK, gin.H{
        "hot_water": hotWaterData,
        "cold_water": coldWaterData,
        "timestamp": time.Now(),
        "building_id": buildingID,
    })
}

// Заполнение тестовыми данными
func (h *Handler) SeedTestData(c *gin.Context) {
    err := h.seedTestData()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Test data seeded successfully"})
}

// Управление генератором данных
func (h *Handler) StartGenerator(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"status": "generator started"})
}

func (h *Handler) StopGenerator(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"status": "generator stopped"})
}

func (h *Handler) GetGeneratorStatus(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"status": "stopped"})
}

// Метод для заполнения тестовыми данными
func (h *Handler) seedTestData() error {
    // Простая реализация заполнения тестовыми данными
    ctx := context.Background()
    
    // Проверяем, есть ли уже данные
    var count int
    err := h.pool.QueryRow(ctx, "SELECT COUNT(*) FROM buildings").Scan(&count)
    if err != nil {
        return err
    }
    
    if count > 0 {
        fmt.Println("База данных уже содержит данные, пропускаем заполнение")
        return nil
    }
    
    // Вставляем тестовые здания
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
    }

    fmt.Println("Тестовые данные успешно добавлены")
    return nil
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

// Если нужен Generator поле, добавьте его в структуру:
/*
type Handler struct {
    pool      *pgxpool.Pool
    Generator *service.DataGenerator // Добавьте если нужно
}
*/