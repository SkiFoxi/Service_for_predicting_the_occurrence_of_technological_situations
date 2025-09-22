package api

import (
    "context"
    "net/http"

    "service/internal/models"
    "service/internal/service"

    "github.com/gin-gonic/gin"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/google/uuid"
)

type Handler struct {
    pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
    return &Handler{pool: pool}
}

func (h *Handler) GetBuildings(c *gin.Context) {
    rows, err := h.pool.Query(context.Background(), `
        SELECT id, address, fias_id, unom_id, created_at, updated_at 
        FROM buildings ORDER BY address`)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer rows.Close()

    var buildings []models.Building
    for rows.Next() {
        var b models.Building
        err := rows.Scan(&b.ID, &b.Address, &b.FiasID, &b.UnomID, &b.CreatedAt, &b.UpdatedAt)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        buildings = append(buildings, b)
    }

    c.JSON(http.StatusOK, buildings)
}

func (h *Handler) AnalyzeBuilding(c *gin.Context) {
    buildingIDStr := c.Param("id")
    buildingID, err := uuid.Parse(buildingIDStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid building ID"})
        return
    }

    analyzer := service.NewAnalyzer(h.pool)
    result, err := analyzer.AnalyzeConsumption(context.Background(), buildingID, 30)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, result)
}

func (h *Handler) SeedTestData(c *gin.Context) {
    err := h.seedTestData()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Test data seeded successfully"})
}

func (h *Handler) seedTestData() error {
    // TODO: Реализовать заполнение тестовыми данными
    return nil
}