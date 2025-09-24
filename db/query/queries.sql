-- name: GetBuildings :many
SELECT * FROM buildings ORDER BY address; 

-- name: GetBuildingByID :one
SELECT * FROM buildings WHERE id = $1;

-- name: GetITPByBuildingID :many
SELECT * FROM itp WHERE building_id = $1;

-- name: GetColdWaterData :many
SELECT * FROM cold_water_meters 
WHERE itp_id IN (SELECT id FROM itp WHERE building_id = $1) 
AND timestamp BETWEEN $2 AND $3
ORDER BY timestamp;

-- name: GetHotWaterData :many
SELECT * FROM hot_water_meters 
WHERE building_id = $1 
AND timestamp BETWEEN $2 AND $3
ORDER BY timestamp;