#!/bin/bash

echo "Applying database migrations..."

# Ждем пока PostgreSQL запустится
until pg_isready -h localhost -p 5434 -U root; do
  echo "Waiting for PostgreSQL to start..."
  sleep 2
done

# Применяем миграции по порядку
for migration in /app/migrations/*.up.sql; do
  echo "Applying migration: $(basename $migration)"
  psql -h localhost -p 5434 -U root -d base_service -f "$migration"
done

echo "Migrations applied successfully!"

# Заполняем начальные данные если нужно
if [ "$FILL_INITIAL_DATA" = "true" ]; then
  echo "Filling initial data..."
  psql -h localhost -p 5434 -U root -d base_service -f /app/scripts/fill_initial_data.sql
fi