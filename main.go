package main

import (
    "log"
    "net/http"

    "service/internal/api"
    "service/internal/database"

    "github.com/gin-gonic/gin"
)

func main() {
    // Конфигурация базы данных
    dbConfig := database.Config{
        Host:     "localhost",
        Port:     "5434",
        User:     "root",
        Password: "secret",
        DBName:   "base_service",
        SSLMode:  "disable",
    }

    // Подключение к базе данных
    pool, err := database.NewPool(dbConfig)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer pool.Close()

    // Создание HTTP сервера
    router := gin.Default()

    // Инициализация обработчиков
    handler := api.NewHandler(pool)

    // Маршруты
    router.GET("/api/buildings", handler.GetBuildings)
    router.GET("/api/analysis/:id", handler.AnalyzeBuilding)
    router.POST("/api/seed-data", handler.SeedTestData)

    // Health check
    router.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status": "ok"})
    })

    // Запуск сервера
    log.Println("Server starting on :8080")
    if err := router.Run(":8080"); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
}