package main

import (
    "context"
    "fmt"
    "log"
    "math/rand"
    "net/http"
    "os"
    "time"

    "service/internal/api"
    "service/internal/database"
    "service/internal/service"

    "github.com/gin-gonic/gin"
    "github.com/gin-contrib/cors"
    "github.com/gorilla/websocket"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/google/uuid"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true // В продакшене нужно ограничить домены
    },
}

var wsConnections = make(map[*websocket.Conn]bool)
var dataGenerator *service.DataGenerator

// Вспомогательная функция для создания тестовых зданий
func createTestBuildings(pool *pgxpool.Pool) error {
    ctx := context.Background()
    
    // Проверяем, есть ли уже здания
    var count int
    err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM buildings").Scan(&count)
    if err != nil {
        return err
    }
    
    if count > 0 {
        log.Printf("Buildings already exist: %d", count)
        return nil
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
        _, err := pool.Exec(ctx, `
            INSERT INTO buildings (id, address, fias_id, unom_id, created_at, updated_at)
            VALUES ($1, $2, $3, $4, NOW(), NOW())`,
            b.id, b.address, b.fiasID, b.unomID)
        if err != nil {
            return fmt.Errorf("insert building %s: %w", b.address, err)
        }
        
        // Создаем ИТП для каждого здания
        itpID := uuid.New()
        _, err = pool.Exec(ctx, `
            INSERT INTO itp (id, itp_number, building_id, created_at, updated_at)
            VALUES ($1, $2, $3, NOW(), NOW())`,
            itpID, fmt.Sprintf("ИТП-%s", b.unomID), b.id)
        if err != nil {
            log.Printf("Warning: failed to create ITP for building %s: %v", b.address, err)
        }
    }

    log.Printf("Created %d test buildings", len(buildings))
    return nil
}

// Вспомогательная функция для генерации исторических данных
func generateHistoricalData(pool *pgxpool.Pool, days int) error {
    ctx := context.Background()
    
    // Получаем все здания
    rows, err := pool.Query(ctx, "SELECT id FROM buildings")
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
    
    log.Printf("Generating historical data for %d buildings over %d days...", len(buildingIDs), days)
    
    for _, buildingID := range buildingIDs {
        // Получаем ITP для здания
        var itpID uuid.UUID
        err = pool.QueryRow(ctx, "SELECT id FROM itp WHERE building_id = $1 LIMIT 1", buildingID).Scan(&itpID)
        if err != nil {
            continue
        }

        // Генерируем данные за каждый день
        for i := 0; i < days; i++ {
            currentTime := baseTime.AddDate(0, 0, i)
            
            // Реалистичные данные для МКД
            hotWaterFlow1 := 2 + rand.Intn(4)
            hotWaterFlow2 := 1 + rand.Intn(3)
            coldWaterFlow := 3 + rand.Intn(7)

            // Данные ГВС
            pool.Exec(ctx, `
                INSERT INTO hot_water_meters (id, building_id, flow_rate_ch1, flow_rate_ch2, timestamp, created_at)
                VALUES ($1, $2, $3, $4, $5, NOW())`,
                uuid.New(), buildingID, hotWaterFlow1, hotWaterFlow2, currentTime)

            // Данные ХВС
            pool.Exec(ctx, `
                INSERT INTO cold_water_meters (id, itp_id, flow_rate, timestamp, created_at)
                VALUES ($1, $2, $3, $4, NOW())`,
                uuid.New(), itpID, coldWaterFlow, currentTime)

            // Температурные данные
            supplyTemp := 65 + rand.Intn(5)
            returnTemp := 42 + rand.Intn(4)
            deltaTemp := supplyTemp - returnTemp
            
            pool.Exec(ctx, `
                INSERT INTO temperature_readings (id, building_id, supply_temp, return_temp, delta_temp, timestamp, created_at)
                VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
                uuid.New(), buildingID, supplyTemp, returnTemp, deltaTemp, currentTime)
        }
    }

    log.Printf("Historical data generation completed for %d days", days)
    return nil
}

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
    dataGenerator = service.NewDataGenerator(pool)

    // Запускаем генерацию данных если включено
    if os.Getenv("ENABLE_DATA_GENERATION") == "true" {
        ctx := context.Background()
        dataGenerator.StartContinuousGeneration(ctx)
        log.Println("Continuous data generation enabled")
    }

    // Заполняем начальные данные если нужно
    if os.Getenv("FILL_INITIAL_DATA") == "true" {
        // Создаем тестовые здания
        err := createTestBuildings(pool)
        if err != nil {
            log.Printf("Warning: could not create test buildings: %v", err)
        } else {
            log.Println("Test buildings created successfully")
        }
        
        // Заполняем историческими данными
        err = generateHistoricalData(pool, 7) // 7 дней данных
        if err != nil {
            log.Printf("Warning: could not fill initial data: %v", err)
        } else {
            log.Println("Initial data filled successfully")
        }
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

    // Главная страница
    router.GET("/", func(c *gin.Context) {
        c.HTML(http.StatusOK, "index.html", nil)
    })

    // WebSocket endpoint
    router.GET("/ws", handleWebSocket)

    // API маршруты
    apiGroup := router.Group("/api")
    {
        apiGroup.GET("/buildings", handler.GetBuildings)
        apiGroup.GET("/buildings/:id", handler.GetBuildingByID)
        apiGroup.GET("/analysis/:id", handler.AnalyzeBuilding)
        apiGroup.POST("/seed-data", handler.SeedTestData)
        apiGroup.GET("/realtime/:id", handler.GetRealtimeData)
        apiGroup.POST("/generator/start", handler.StartGenerator)
        apiGroup.POST("/generator/stop", handler.StopGenerator)
        apiGroup.GET("/generator/status", handler.GetGeneratorStatus)
        apiGroup.GET("/debug/:id", handler.DebugData)
        apiGroup.POST("/generate-history", handler.GenerateHistory)
        apiGroup.POST("/create-test-buildings", handler.CreateTestBuildings)
        apiGroup.POST("/generate-complete-history", handler.GenerateCompleteHistoricalData)
        
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
            
            err := pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM buildings").Scan(&buildingCount)
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
                "generator_running": dataGenerator != nil && dataGenerator.IsRunning(),
                "websocket_connections": len(wsConnections),
                "timestamp": time.Now().Format(time.RFC3339),
            })
        })
    }

    // Запуск сервера
    log.Println("Server starting on :8080")
    log.Println("Available endpoints:")
    log.Println("  http://localhost:8080/ - Frontend")
    log.Println("  http://localhost:8080/ws - WebSocket")
    log.Println("  http://localhost:8080/api/buildings - Buildings API")
    log.Println("  http://localhost:8080/api/test - Test API")
    log.Println("  http://localhost:8080/api/health - Health check")
    log.Println("  http://localhost:8080/api/realtime/:id - Real-time data")
    log.Println("  http://localhost:8080/api/analysis/:id - Intelligent analysis")
    
    if err := router.Run(":8080"); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
}

func handleWebSocket(c *gin.Context) {
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        log.Printf("WebSocket upgrade error: %v", err)
        return
    }
    defer conn.Close()

    wsConnections[conn] = true
    log.Printf("WebSocket client connected. Total connections: %d", len(wsConnections))

    // Отправляем начальные данные
    conn.WriteJSON(gin.H{
        "type": "connected",
        "message": "WebSocket connected successfully",
        "timestamp": time.Now().Format(time.RFC3339),
    })

    // Обработка сообщений от клиента
    for {
        var message map[string]interface{}
        err := conn.ReadJSON(&message)
        if err != nil {
            log.Printf("WebSocket read error: %v", err)
            delete(wsConnections, conn)
            log.Printf("WebSocket client disconnected. Total connections: %d", len(wsConnections))
            break
        }

        // Обработка команд от клиента
        if msgType, ok := message["type"].(string); ok {
            switch msgType {
            case "ping":
                conn.WriteJSON(gin.H{
                    "type": "pong",
                    "timestamp": time.Now().Format(time.RFC3339),
                })
            case "subscribe":
                conn.WriteJSON(gin.H{
                    "type": "subscribed",
                    "message": "Subscribed to realtime updates",
                    "channels": message["channels"],
                })
            }
        }
    }
}

// Функция для рассылки обновлений всем подключенным клиентам
func broadcastUpdate(data interface{}) {
    for conn := range wsConnections {
        err := conn.WriteJSON(data)
        if err != nil {
            conn.Close()
            delete(wsConnections, conn)
            log.Printf("Removed disconnected WebSocket client. Total connections: %d", len(wsConnections))
        }
    }
}

// Функция для отправки данных реального времени
func broadcastRealtimeData(buildingID string, data interface{}) {
    updateMessage := gin.H{
        "type": "realtime_update",
        "building_id": buildingID,
        "data": data,
        "timestamp": time.Now().Format(time.RFC3339),
        "update_id": time.Now().Unix(),
    }
    broadcastUpdate(updateMessage)
}