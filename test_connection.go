package main

import (
	"context"
	"fmt"
	"log"
	"github.com/jackc/pgx/v5/pgxpool"
	"service/internal/database"
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

	fmt.Printf("Параметры подключения:\n")
	fmt.Printf("   Host: %s\n", dbConfig.Host)
	fmt.Printf("   Port: %s\n", dbConfig.Port)
	fmt.Printf("   User: %s\n", dbConfig.User)
	fmt.Printf("   DBName: %s\n", dbConfig.DBName)

	// Подключение к базе данных
	pool, err := database.NewPool(dbConfig)
	if err != nil {
		log.Fatalf("Ошибка подключения: %v", err)
	}
	defer pool.Close()

	fmt.Println("Подключение успешно установлено!")

	// Проверяем, что база данных доступна
	err = testConnection(pool)
	if err != nil {
		log.Fatalf("Ошибка при проверке соединения: %v", err)
	}

	fmt.Println("Проверка соединения прошла успешно!")
}

func testConnection(pool *pgxpool.Pool) error {
	// Получаем соединение из пула
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		return fmt.Errorf("ошибка получения соединения: %w", err)
	}
	defer conn.Release()

	// Простой запрос для проверки соединения
	var version string
	err = conn.QueryRow(context.Background(), "SELECT version()").Scan(&version)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса: %w", err)
	}

	fmt.Printf("Версия PostgreSQL: %s\n", version)

	// Проверяем существование таблиц
	var tableCount int
	err = conn.QueryRow(context.Background(), `
		SELECT COUNT(*) 
		FROM information_schema.tables 
		WHERE table_schema = 'public'
	`).Scan(&tableCount)
	if err != nil {
		return fmt.Errorf("ошибка проверки таблиц: %w", err)
	}

	fmt.Printf("Количество таблиц в базе: %d\n", tableCount)

	// Если есть таблицы, покажем их список
	if tableCount > 0 {
		rows, err := conn.Query(context.Background(), `
			SELECT table_name 
			FROM information_schema.tables 
			WHERE table_schema = 'public'
			ORDER BY table_name
		`)
		if err != nil {
			return fmt.Errorf("ошибка получения списка таблиц: %w", err)
		}
		defer rows.Close()

		fmt.Println("Список таблиц:")
		for rows.Next() {
			var tableName string
			err := rows.Scan(&tableName)
			if err != nil {
				return fmt.Errorf("ошибка чтения названия таблицы: %w", err)
			}
			fmt.Printf("   - %s\n", tableName)
		}
	}

	return nil
}