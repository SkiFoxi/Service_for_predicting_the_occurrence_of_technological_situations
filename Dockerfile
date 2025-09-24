FROM golang:1.23-alpine

WORKDIR /app

# Копируем go модули сначала для кэширования
COPY go.mod go.sum ./
RUN go mod download

# Копируем ВЕСЬ проект
COPY . .

# Создаем структуру папок в контейнере
RUN mkdir -p /app/db /app/internal /app/scripts /app/web

# Копируем папки с содержимым
COPY db/ /app/db/
COPY internal/ /app/internal/
COPY scripts/ /app/scripts/
COPY web/ /app/web/

# Удаляем дублирующийся файл если он есть
RUN rm -f /app/web/fill_initial_data.sql

# Даем права на выполнение скриптов
RUN chmod +x /app/scripts/apply_migrations.sh

# Собираем приложение
RUN go build -o main .

EXPOSE 8080

CMD ["./main"]