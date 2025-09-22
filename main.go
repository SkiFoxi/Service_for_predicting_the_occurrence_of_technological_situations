package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
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

    // Настройка CORS для работы с фронтендом
    router.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"*"}, // Разрешаем все origins для разработки
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: true,
        MaxAge: 12 * time.Hour,
    }))

    // Правильно настраиваем статические файлы
    router.Static("/css", "./web/css")
    router.Static("/js", "./web/js")
    router.Static("/assets", "./web/assets")
    
    // HTML файлы
    router.LoadHTMLFiles("./web/index.html")

    // Инициализация обработчиков
    handler := api.NewHandler(pool)
    handler.Generator = dataGenerator

    // Главная страница
    router.GET("/", func(c *gin.Context) {
        c.HTML(http.StatusOK, "index.html", nil)
    })

    // API маршруты
    apiGroup := router.Group("/api")
    {
        apiGroup.GET("/buildings", handler.GetBuildings)
        apiGroup.GET("/buildings/:id", handler.GetBuildingByID)
        apiGroup.GET("/analysis/:id", handler.AnalyzeBuilding)
        apiGroup.POST("/seed-data", handler.SeedTestData)
        apiGroup.GET("/realtime/:id", handler.GetRealtimeData)
        
        // Управление генерацией данных
        apiGroup.POST("/generator/start", handler.StartGenerator)
        apiGroup.POST("/generator/stop", handler.StopGenerator)
        apiGroup.GET("/generator/status", handler.GetGeneratorStatus)
    }

    // Health check
    router.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "status":    "ok",
            "database":  "connected",
            "timestamp": time.Now().Format(time.RFC3339),
        })
    })

    // Обработка graceful shutdown
    setupGracefulShutdown(dataGenerator)

    // Запуск сервера
    log.Println("Server starting on :8080")
    log.Println("Frontend available at: http://localhost:8080")
    log.Println("API available at: http://localhost:8080/api")
    
    if err := router.Run(":8080"); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
}

func setupGracefulShutdown(generator *service.DataGenerator) {
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        <-quit
        log.Println("Shutting down server...")
        if generator != nil {
            generator.Stop()
        }
        os.Exit(0)
    }()
}