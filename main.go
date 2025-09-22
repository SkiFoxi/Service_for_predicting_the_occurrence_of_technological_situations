package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "time"

    "service/internal/api"
    "service/internal/database"
    "service/internal/service"

    "github.com/gin-gonic/gin"
    "github.com/gin-contrib/cors"
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

    // Создаем генератор данных
    dataGenerator := service.NewDataGenerator(pool)
    
    // Загружаем существующие данные
    ctx := context.Background()
    if err := dataGenerator.LoadExistingData(ctx); err != nil {
        log.Printf("Warning: could not load existing data: %v", err)
    }

    // Запускаем генерацию данных (если включено)
    if os.Getenv("ENABLE_DATA_GENERATION") == "true" {
        dataGenerator.Start(ctx)
        defer dataGenerator.Stop()
        log.Println("Real-time data generation enabled")
    }

    // Создание HTTP сервера
    router := gin.Default()

    // Настройка CORS
    router.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"*"},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: true,
        MaxAge: 12 * time.Hour,
    }))

    // Статические файлы
    router.Static("/css", "./web/css")
    router.Static("/js", "./web/js")
    router.Static("/assets", "./web/assets")
    
    // HTML файлы
    router.LoadHTMLGlob("./web/*.html")

    // Инициализация обработчиков
    handler := api.NewHandler(pool)
    // Убираем строку с Generator, так как его нет в Handler

    // Главная страница
    router.GET("/", func(c *gin.Context) {
        c.HTML(http.StatusOK, "index.html", nil)
    })

    // API маршруты
    apiGroup := router.Group("/api")
    {
        apiGroup.GET("/buildings", handler.GetBuildings)
        // Убираем пока этот маршрут, так как метода нет
        // apiGroup.GET("/buildings/:id", handler.GetBuildingByID)
        apiGroup.GET("/analysis/:id", handler.AnalyzeBuilding)
        apiGroup.POST("/seed-data", handler.SeedTestData)
        apiGroup.GET("/realtime/:id", handler.GetRealtimeData)
        apiGroup.POST("/generator/start", handler.StartGenerator)
        apiGroup.POST("/generator/stop", handler.StopGenerator)
        apiGroup.GET("/generator/status", handler.GetGeneratorStatus)
        
        // Простой тестовый эндпоинт
        apiGroup.GET("/test", func(c *gin.Context) {
            c.JSON(http.StatusOK, gin.H{
                "status": "ok", 
                "message": "API is working",
                "timestamp": time.Now().Format(time.RFC3339),
            })
        })
        
        // Health check
        apiGroup.GET("/health", func(c *gin.Context) {
            // Проверяем соединение с БД
            var dbStatus string
            var buildingCount int
            
            err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM buildings").Scan(&buildingCount)
            if err != nil {
                dbStatus = "error: " + err.Error()
                buildingCount = 0
            } else {
                dbStatus = "connected"
            }
            
            c.JSON(http.StatusOK, gin.H{
                "status": "ok",
                "database": dbStatus,
                "buildings_count": buildingCount,
                "timestamp": time.Now().Format(time.RFC3339),
            })
        })
    }

    // Запуск сервера
    log.Println("Server starting on :8080")
    log.Println("Available endpoints:")
    log.Println("  http://localhost:8080/ - Frontend")
    log.Println("  http://localhost:8080/api/buildings - Buildings API")
    log.Println("  http://localhost:8080/api/test - Test API")
    log.Println("  http://localhost:8080/api/health - Health check")
    log.Println("  http://localhost:8080/api/realtime/:id - Real-time data")
    
    if err := router.Run(":8080"); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
}