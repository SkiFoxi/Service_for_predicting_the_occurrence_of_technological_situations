package api

import (
    "context"
    "net/http"
    "time"

    "service/internal/service"

    "github.com/gin-gonic/gin"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/google/uuid"
)

type Handler struct {
    pool      *pgxpool.Pool
    Generator *service.DataGenerator
}

func NewHandler(pool *pgxpool.Pool) *Handler {
    return &Handler{pool: pool}
}

// Получение всех зданий
func (h *Handler) GetBuildings(c *gin.Context) {
    rows, err := h.pool.Query(context.Background(), `
        SELECT id, address, fias_id, unom_id, created_at, updated_at 
        FROM buildings ORDER BY address`)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer rows.Close()

    var buildings []struct {
        ID        uuid.UUID `json:"id"`
        Address   string    `json:"address"`
        FiasID    string    `json:"fias_id"`
        UnomID    string    `json:"unom_id"`
        CreatedAt time.Time `json:"created_at"`
        UpdatedAt time.Time `json:"updated_at"`
    }

    for rows.Next() {
        var b struct {
            ID        uuid.UUID `json:"id"`
            Address   string    `json:"address"`
            FiasID    string    `json:"fias_id"`
            UnomID    string    `json:"unom_id"`
            CreatedAt time.Time `json:"created_at"`
            UpdatedAt time.Time `json:"updated_at"`
        }
        err := rows.Scan(&b.ID, &b.Address, &b.FiasID, &b.UnomID, &b.CreatedAt, &b.UpdatedAt)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        buildings = append(buildings, b)
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

    var building struct {
        ID        uuid.UUID `json:"id"`
        Address   string    `json:"address"`
        FiasID    string    `json:"fias_id"`
        UnomID    string    `json:"unom_id"`
        CreatedAt time.Time `json:"created_at"`
        UpdatedAt time.Time `json:"updated_at"`
    }

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

// Анализ потребления
func (h *Handler) AnalyzeBuilding(c *gin.Context) {
    buildingIDStr := c.Param("id")
    buildingID, err := uuid.Parse(buildingIDStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid building ID"})
        return
    }

    days := 30 // по умолчанию 30 дней

    analyzer := service.NewAnalyzer(h.pool)
    result, err := analyzer.AnalyzeConsumption(context.Background(), buildingID, days)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
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
        ORDER BY timestamp DESC 
        LIMIT 1`, buildingID).Scan(
        &hotWaterData.FlowRateCh1, &hotWaterData.FlowRateCh2, &hotWaterData.Timestamp)

    if err != nil {
        // Если нет данных, возвращаем нулевые значения
        hotWaterData = struct {
            FlowRateCh1 int       `json:"flow_rate_ch1"`
            FlowRateCh2 int       `json:"flow_rate_ch2"`
            Timestamp   time.Time `json:"timestamp"`
        }{0, 0, time.Now()}
    }

    // Получаем последние данные по ХВС (через ИТП)
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

    if err != nil {
        coldWaterData.TotalFlowRate = 0
        coldWaterData.Timestamp = time.Now()
    }

    c.JSON(http.StatusOK, gin.H{
        "hot_water": hotWaterData,
        "cold_water": coldWaterData,
        "timestamp": time.Now(),
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
    if h.Generator == nil {
        h.Generator = service.NewDataGenerator(h.pool)
        if err := h.Generator.LoadExistingData(c.Request.Context()); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
    }
    
    h.Generator.Start(c.Request.Context())
    c.JSON(http.StatusOK, gin.H{"status": "generator started"})
}

func (h *Handler) StopGenerator(c *gin.Context) {
    if h.Generator != nil {
        h.Generator.Stop()
        c.JSON(http.StatusOK, gin.H{"status": "generator stopped"})
    } else {
        c.JSON(http.StatusBadRequest, gin.H{"error": "generator not running"})
    }
}

func (h *Handler) GetGeneratorStatus(c *gin.Context) {
    status := "stopped"
    if h.Generator != nil && h.Generator.IsRunning() {
        status = "running"
    }
    
    c.JSON(http.StatusOK, gin.H{"status": status})
}

// Метод для заполнения тестовыми данными
func (h *Handler) seedTestData() error {
    // Реализация заполнения тестовыми данными
    // (код из предыдущего ответа)
    return nil
}