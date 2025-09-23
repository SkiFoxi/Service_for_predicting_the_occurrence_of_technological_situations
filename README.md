# Service_for_predicting_the_occurrence_of_technological_situations
Использовать генератор рандомных значений: ENABLE_DATA_GENERATION=true go run main.go
Без него запуск программы: go run main.go

# Запуск с непрерывной генерацией данных
ENABLE_DATA_GENERATION=true go run main.go

# Или отдельно запустить генератор через API
curl -X POST http://localhost:8080/api/generator/start