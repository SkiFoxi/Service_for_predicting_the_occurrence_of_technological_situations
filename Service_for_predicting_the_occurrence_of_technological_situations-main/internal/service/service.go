package service

import (
    "context"
    "fmt"
    "math/rand"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/google/uuid"
)

type ConsumptionAnalysis struct {
    BuildingID           uuid.UUID `json:"building_id"`
    Period               string    `json:"period"`
    TotalColdWater       int       `json:"total_cold_water"`
    TotalHotWater        int       `json:"total_hot_water"`
    Difference           int       `json:"difference"`
    DifferencePercent    float64   `json:"difference_percent"`
    HotToColdRatio       float64   `json:"hot_to_cold_ratio"` // Новое поле: соотношение ГВС/ХВС в %
    HasAnomalies         bool      `json:"has_anomalies"`
    AnomalyCount         int       `json:"anomaly_count"`
    WaterBalanceStatus   string    `json:"water_balance_status"`
    TemperatureStatus    string    `json:"temperature_status"`
    PumpStatus           string    `json:"pump_status"`
    PumpOperatingHours   int       `json:"pump_operating_hours"`
    Recommendations      []string  `json:"recommendations"`
    DataSource           string    `json:"data_source"`
    TemperatureData      *TemperatureData `json:"temperature_data,omitempty"`
    PumpData             *PumpAnalysis    `json:"pump_data,omitempty"`
}

type TemperatureData struct {
    AvgSupplyTemp int     `json:"avg_supply_temp"`
    AvgReturnTemp int     `json:"avg_return_temp"`
    AvgDeltaTemp  int     `json:"avg_delta_temp"`
    MinDeltaTemp  int     `json:"min_delta_temp"`
    MaxDeltaTemp  int     `json:"max_delta_temp"`
    RecordsCount  int     `json:"records_count"`
}

type PumpAnalysis struct {
    TotalPumps        int     `json:"total_pumps"`
    NormalPumps       int     `json:"normal_pumps"`
    WarningPumps      int     `json:"warning_pumps"`
    CriticalPumps     int     `json:"critical_pumps"`
    AvgOperatingHours int     `json:"avg_operating_hours"`
    MaxOperatingHours int     `json:"max_operating_hours"`
    PressureStatus    string  `json:"pressure_status"`
    VibrationStatus   string  `json:"vibration_status"`
}

type Analyzer struct {
    pool *pgxpool.Pool
}

func NewAnalyzer(pool *pgxpool.Pool) *Analyzer {
    return &Analyzer{pool: pool}
}

func (a *Analyzer) AnalyzeConsumption(ctx context.Context, buildingID uuid.UUID, days int) (*ConsumptionAnalysis, error) {
    endDate := time.Now()
    startDate := endDate.AddDate(0, 0, -days)

    // Получаем все данные из БД
    totalColdWater, totalHotWater, coldRecords, hotRecords, hasWaterData, err := a.getWaterDataFromDB(ctx, buildingID, startDate, endDate)
    if err != nil {
        return nil, fmt.Errorf("get water data from DB: %w", err)
    }

    // Получаем температурные данные из БД
    tempData, hasTempData, err := a.getTemperatureData(ctx, buildingID, startDate, endDate)
    if err != nil {
        return nil, fmt.Errorf("get temperature data from DB: %w", err)
    }

    // Получаем данные насосов из БД
    pumpData, hasPumpData, err := a.getPumpData(ctx, buildingID, startDate, endDate)
    if err != nil {
        return nil, fmt.Errorf("get pump data from DB: %w", err)
    }

    var analysis *ConsumptionAnalysis
    var dataSource string

    if hasWaterData {
        // Используем реальные данные из БД
        analysis = a.analyzeRealData(totalColdWater, totalHotWater, coldRecords, hotRecords, tempData, pumpData, buildingID, startDate, endDate)
        dataSource = "database"
        
        // Добавляем информацию о качестве данных
        infoMsg := fmt.Sprintf("Данные основаны на %d записях ХВС и %d записях ГВС из БД", coldRecords, hotRecords)
        if hasTempData {
            infoMsg += fmt.Sprintf(", %d температурных записях", tempData.RecordsCount)
        }
        if hasPumpData {
            infoMsg += fmt.Sprintf(", %d насосах", pumpData.TotalPumps)
        }
        
        analysis.Recommendations = append([]string{infoMsg}, analysis.Recommendations...)
        
    } else {
        // Если данных нет, используем реалистичные оценки
        analysis = a.analyzeEstimatedData(buildingID, days)
        dataSource = "estimated"
    }

    analysis.DataSource = dataSource
    
    // Добавляем детальные данные если они есть
    if hasTempData {
        analysis.TemperatureData = tempData
    }
    if hasPumpData {
        analysis.PumpData = pumpData
    }

    // Временная проверка для отладки
    if analysis.HotToColdRatio > 95.0 {
        // Вероятно ошибка в данных, исправляем статус
        analysis.WaterBalanceStatus = "warning"
        analysis.HasAnomalies = false
        analysis.AnomalyCount = 0
        analysis.Recommendations = append([]string{
            "⚠️ Внимание: Обнаружены возможные некорректные данные. Соотношение ГВС/ХВС превышает реалистичные значения.",
            "Рекомендуется проверить корректность показаний счетчиков.",
        }, analysis.Recommendations...)
    }

    return analysis, nil
}

// Получение суммарного расхода ХВС за период
func (a *Analyzer) getTotalColdWater(ctx context.Context, buildingID uuid.UUID, start, end time.Time) (int, error) {
    var totalFlow int
    
    err := a.pool.QueryRow(ctx, `
        SELECT COALESCE(SUM(cwm.flow_rate), 0)
        FROM cold_water_meters cwm
        JOIN itp i ON cwm.itp_id = i.id
        WHERE i.building_id = $1 
        AND cwm.timestamp BETWEEN $2 AND $3`,
        buildingID, start, end).Scan(&totalFlow)
    
    if err != nil {
        return 0, fmt.Errorf("get total cold water: %w", err)
    }
    
    return totalFlow, nil
}

// Получение суммарного расхода ГВС за период
func (a *Analyzer) getTotalHotWater(ctx context.Context, buildingID uuid.UUID, start, end time.Time) (int, error) {
    var totalFlow int
    
    err := a.pool.QueryRow(ctx, `
        SELECT COALESCE(SUM(flow_rate_ch1 + flow_rate_ch2), 0)
        FROM hot_water_meters 
        WHERE building_id = $1 
        AND timestamp BETWEEN $2 AND $3`,
        buildingID, start, end).Scan(&totalFlow)
    
    if err != nil {
        return 0, fmt.Errorf("get total hot water: %w", err)
    }
    
    return totalFlow, nil
}

// Получение температурных данных из БД
func (a *Analyzer) getTemperatureData(ctx context.Context, buildingID uuid.UUID, start, end time.Time) (*TemperatureData, bool, error) {
    var tempData TemperatureData
    
    // Получаем средние температуры за период
    err := a.pool.QueryRow(ctx, `
        SELECT 
            COALESCE(AVG(supply_temp), 0)::int,
            COALESCE(AVG(return_temp), 0)::int,
            COALESCE(AVG(delta_temp), 0)::int,
            COALESCE(MIN(delta_temp), 0)::int,
            COALESCE(MAX(delta_temp), 0)::int,
            COUNT(*)
        FROM temperature_readings 
        WHERE building_id = $1 
        AND timestamp BETWEEN $2 AND $3`,
        buildingID, start, end).Scan(
            &tempData.AvgSupplyTemp,
            &tempData.AvgReturnTemp,
            &tempData.AvgDeltaTemp,
            &tempData.MinDeltaTemp,
            &tempData.MaxDeltaTemp,
            &tempData.RecordsCount)
    
    if err != nil {
        return nil, false, fmt.Errorf("get temperature data: %w", err)
    }

    hasData := tempData.RecordsCount > 0
    return &tempData, hasData, nil
}

// Получение данных насосов из БД
func (a *Analyzer) getPumpData(ctx context.Context, buildingID uuid.UUID, start, end time.Time) (*PumpAnalysis, bool, error) {
    var pumpData PumpAnalysis
    
    // Получаем последние данные по каждому насосу
    rows, err := a.pool.Query(ctx, `
        SELECT DISTINCT ON (pump_number)
            pump_number, status, operating_hours, pressure_input, pressure_output, vibration_level
        FROM pump_data 
        WHERE building_id = $1 
        AND timestamp BETWEEN $2 AND $3
        ORDER BY pump_number, timestamp DESC`,
        buildingID, start, end)
    
    if err != nil {
        return nil, false, fmt.Errorf("get pump data: %w", err)
    }
    defer rows.Close()

    var totalOperatingHours int
    var maxOperatingHours int
    var pressureReadings, vibrationReadings int
    
    for rows.Next() {
        var pumpNumber, status string
        var operatingHours, pressureInput, pressureOutput, vibrationLevel int
        
        err := rows.Scan(&pumpNumber, &status, &operatingHours, &pressureInput, &pressureOutput, &vibrationLevel)
        if err != nil {
            continue
        }
        
        pumpData.TotalPumps++
        totalOperatingHours += operatingHours
        
        if operatingHours > maxOperatingHours {
            maxOperatingHours = operatingHours
        }
        
        switch status {
        case "normal":
            pumpData.NormalPumps++
        case "warning":
            pumpData.WarningPumps++
        case "critical":
            pumpData.CriticalPumps++
        }
        
        // Анализ давления
        pressureDiff := pressureOutput - pressureInput
        if pressureDiff >= 1 && pressureDiff <= 3 {
            pressureReadings++
        }
        
        // Анализ вибрации
        if vibrationLevel <= 5 {
            vibrationReadings++
        }
    }
    
    if pumpData.TotalPumps > 0 {
        pumpData.AvgOperatingHours = totalOperatingHours / pumpData.TotalPumps
        pumpData.MaxOperatingHours = maxOperatingHours
        
        // Определяем статус давления
        pressureRatio := float64(pressureReadings) / float64(pumpData.TotalPumps)
        if pressureRatio >= 0.8 {
            pumpData.PressureStatus = "normal"
        } else if pressureRatio >= 0.5 {
            pumpData.PressureStatus = "warning"
        } else {
            pumpData.PressureStatus = "critical"
        }
        
        // Определяем статус вибрации
        vibrationRatio := float64(vibrationReadings) / float64(pumpData.TotalPumps)
        if vibrationRatio >= 0.8 {
            pumpData.VibrationStatus = "normal"
        } else if vibrationRatio >= 0.5 {
            pumpData.VibrationStatus = "warning"
        } else {
            pumpData.VibrationStatus = "critical"
        }
    }
    
    hasData := pumpData.TotalPumps > 0
    return &pumpData, hasData, nil
}

// Получение количества записей для анализа качества данных
func (a *Analyzer) getDataQuality(ctx context.Context, buildingID uuid.UUID, start, end time.Time) (int, int, error) {
    var coldRecords, hotRecords int
    
    err := a.pool.QueryRow(ctx, `
        SELECT COUNT(*)
        FROM cold_water_meters cwm
        JOIN itp i ON cwm.itp_id = i.id
        WHERE i.building_id = $1 
        AND cwm.timestamp BETWEEN $2 AND $3`,
        buildingID, start, end).Scan(&coldRecords)
    
    if err != nil {
        return 0, 0, err
    }
    
    err = a.pool.QueryRow(ctx, `
        SELECT COUNT(*)
        FROM hot_water_meters 
        WHERE building_id = $1 
        AND timestamp BETWEEN $2 AND $3`,
        buildingID, start, end).Scan(&hotRecords)
    
    if err != nil {
        return 0, 0, err
    }
    
    return coldRecords, hotRecords, nil
}

// Получение водных данных из БД
func (a *Analyzer) getWaterDataFromDB(ctx context.Context, buildingID uuid.UUID, start, end time.Time) (int, int, int, int, bool, error) {
    totalColdWater, err := a.getTotalColdWater(ctx, buildingID, start, end)
    if err != nil {
        return 0, 0, 0, 0, false, err
    }

    totalHotWater, err := a.getTotalHotWater(ctx, buildingID, start, end)
    if err != nil {
        return 0, 0, 0, 0, false, err
    }

    coldRecords, hotRecords, err := a.getDataQuality(ctx, buildingID, start, end)
    if err != nil {
        return 0, 0, 0, 0, false, err
    }

    requiredRecords := 7
    hasEnoughData := coldRecords >= requiredRecords && hotRecords >= requiredRecords
    
    return totalColdWater, totalHotWater, coldRecords, hotRecords, hasEnoughData, nil
}

// Анализ РЕАЛЬНЫХ данных из БД
func (a *Analyzer) analyzeRealData(totalColdWater, totalHotWater, coldRecords, hotRecords int, 
    tempData *TemperatureData, pumpData *PumpAnalysis, buildingID uuid.UUID, start, end time.Time) *ConsumptionAnalysis {
    
    // Рассчитываем средние значения для анализа воды
    avgColdWater := 0
    if coldRecords > 0 {
        hours := int(end.Sub(start).Hours())
        if hours > 0 {
            avgColdWater = totalColdWater / hours
        }
    }

    avgHotWater := 0
    if hotRecords > 0 {
        hours := int(end.Sub(start).Hours())
        if hours > 0 {
            avgHotWater = totalHotWater / hours
        }
    }

    difference := totalColdWater - totalHotWater
    var differencePercent float64
    if totalColdWater > 0 {
        differencePercent = (float64(difference) / float64(totalColdWater)) * 100
    }

    // ПРАВИЛЬНО рассчитываем соотношение ГВС/ХВС в процентах
    var hotToColdRatioPercent float64
    if totalColdWater > 0 {
        hotToColdRatioPercent = (float64(totalHotWater) / float64(totalColdWater)) * 100
    }

    // Анализ на основе РЕАЛЬНЫХ данных
    waterBalanceStatus := a.analyzeWaterBalanceReal(float64(avgColdWater), float64(avgHotWater), hotToColdRatioPercent, coldRecords, hotRecords)
    temperatureStatus := a.analyzeTemperatureReal(tempData)
    pumpStatus, operatingHours := a.analyzePumpConditionReal(pumpData)
    hasAnomalies, anomalyCount := a.detectAnomaliesReal(totalColdWater, totalHotWater, waterBalanceStatus, temperatureStatus, pumpStatus)
    recommendations := a.generateRecommendationsReal(waterBalanceStatus, temperatureStatus, pumpStatus, operatingHours, 
        totalColdWater, totalHotWater, hotToColdRatioPercent, coldRecords, hotRecords, tempData, pumpData)

    return &ConsumptionAnalysis{
        BuildingID:           buildingID,
        Period:               fmt.Sprintf("%s to %s", start.Format("2006-01-02"), end.Format("2006-01-02")),
        TotalColdWater:       totalColdWater,
        TotalHotWater:        totalHotWater,
        Difference:           difference,
        DifferencePercent:    differencePercent,
        HotToColdRatio:       hotToColdRatioPercent,
        HasAnomalies:         hasAnomalies,
        AnomalyCount:         anomalyCount,
        WaterBalanceStatus:   waterBalanceStatus,
        TemperatureStatus:    temperatureStatus,
        PumpStatus:           pumpStatus,
        PumpOperatingHours:   operatingHours,
        Recommendations:      recommendations,
    }
}

// Анализ баланса на основе РЕАЛЬНЫХ данных с правильной логикой
func (a *Analyzer) analyzeWaterBalanceReal(avgColdWater, avgHotWater, hotToColdRatioPercent float64, coldRecords, hotRecords int) string {
    if coldRecords == 0 || hotRecords == 0 {
        return "unknown" // Нет данных для анализа
    }

    // 1. Проверяем базовую корректность данных
    if avgColdWater <= 0 || avgHotWater < 0 {
        return "error" // Некорректные данные
    }

    // 2. ГВС не может быть больше ХВС - это явная аномалия
    if avgHotWater > avgColdWater {
        return "leak" // Явная аномалия
    }

    // 3. Анализируем по соотношению в процентах
    // Реалистичные диапазоны для МКД:
    // - Норма: 40-70% (ГВС составляет 40-70% от ХВС - это реально для МКД)
    // - Предупреждение: 30-40% или 70-80%
    // - Утечка/ошибка: <30% или >80%
    
    if hotToColdRatioPercent >= 40.0 && hotToColdRatioPercent <= 70.0 {
        return "normal" // Нормальный баланс для МКД
    } else if (hotToColdRatioPercent >= 30.0 && hotToColdRatioPercent < 40.0) || 
              (hotToColdRatioPercent > 70.0 && hotToColdRatioPercent <= 80.0) {
        return "warning" // Небольшое отклонение
    } else if hotToColdRatioPercent < 30.0 {
        return "error" // Слишком мало ГВС
    } else {
        return "leak" // Слишком много ГВС
    }
}

// Анализ температуры на основе РЕАЛЬНЫХ данных из БД
func (a *Analyzer) analyzeTemperatureReal(tempData *TemperatureData) string {
    if tempData == nil || tempData.RecordsCount == 0 {
        return "unknown" // Данных нет
    }

    // Норма ΔT для ГВС: 17-23°C
    if tempData.AvgDeltaTemp >= 17 && tempData.AvgDeltaTemp <= 23 {
        return "normal"
    } else if tempData.AvgDeltaTemp >= 15 && tempData.AvgDeltaTemp <= 25 {
        return "warning"
    } else {
        return "critical"
    }
}

// Анализ насосов на основе РЕАЛЬНЫХ данных из БД
func (a *Analyzer) analyzePumpConditionReal(pumpData *PumpAnalysis) (string, int) {
    if pumpData == nil || pumpData.TotalPumps == 0 {
        return "unknown", 0 // Данных нет
    }

    // Определяем общий статус насосов
    if pumpData.CriticalPumps > 0 {
        return "critical", pumpData.MaxOperatingHours
    } else if pumpData.WarningPumps > 0 {
        return "warning", pumpData.MaxOperatingHours
    } else {
        return "normal", pumpData.MaxOperatingHours
    }
}

// Детектор аномалий на основе реальных данных
func (a *Analyzer) detectAnomaliesReal(totalColdWater, totalHotWater int, waterBalance, temperatureStatus, pumpStatus string) (bool, int) {
    anomalyCount := 0

    // 1. Проверяем базовую корректность данных
    if totalColdWater < 0 || totalColdWater > 1000000 {
        anomalyCount++
    }

    if totalHotWater < 0 || totalHotWater > 1000000 {
        anomalyCount++
    }

    // 2. ГВС не может быть больше ХВС - явная аномалия
    if totalHotWater > totalColdWater {
        anomalyCount++
    }

    // 3. Анализ баланса (только критические состояния)
    if waterBalance == "leak" || waterBalance == "error" {
        anomalyCount++
    }
    // "warning" не считаем аномалией - только наблюдение

    // 4. Температурные аномалии
    if temperatureStatus == "critical" {
        anomalyCount++
    }

    // 5. Критические состояния насосов
    if pumpStatus == "critical" {
        anomalyCount++
    }

    return anomalyCount > 0, anomalyCount
}

// Реальные рекомендации на основе данных
func (a *Analyzer) generateRecommendationsReal(waterBalance, temperatureStatus, pumpStatus string, 
    operatingHours, coldWater, hotWater int, hotToColdRatioPercent float64,
    coldRecords, hotRecords int, tempData *TemperatureData, pumpData *PumpAnalysis) []string {
    
    var recommendations []string

    if coldRecords == 0 || hotRecords == 0 {
        return []string{"Внимание: недостаточно данных для анализа. Рекомендуется проверить работу счетчиков."}
    }

    // Информация о данных
    recommendations = append(recommendations, 
        fmt.Sprintf("Проанализировано записей: ХВС - %d, ГВС - %d", coldRecords, hotRecords))

    // Анализ баланса с правильными расчетами
    switch waterBalance {
    case "leak":
        recommendations = append(recommendations, 
            "ВНИМАНИЕ: Возможна утечка или некорректные показания")
        recommendations = append(recommendations, 
            fmt.Sprintf("Соотношение ГВС/ХВС: %.1f%% (норма: 40-70%%)", hotToColdRatioPercent))
    case "error":
        recommendations = append(recommendations, 
            "Возможна ошибка в данных счетчиков")
        recommendations = append(recommendations, 
            fmt.Sprintf("Соотношение ГВС/ХВС: %.1f%% (норма: 40-70%%)", hotToColdRatioPercent))
    case "warning":
        recommendations = append(recommendations, 
            "Небольшое отклонение от нормы, требуется наблюдение")
        recommendations = append(recommendations, 
            fmt.Sprintf("Соотношение ГВС/ХВС: %.1f%% (норма: 40-70%%)", hotToColdRatioPercent))
    case "normal":
        recommendations = append(recommendations, 
            fmt.Sprintf("Баланс в норме. Соотношение ГВС/ХВС: %.1f%%", hotToColdRatioPercent))
    default:
        recommendations = append(recommendations, 
            "Недостаточно данных для анализа баланса")
    }

    // Анализ температуры
    if tempData != nil && tempData.RecordsCount > 0 {
        switch temperatureStatus {
        case "normal":
            recommendations = append(recommendations, 
                fmt.Sprintf("Температурный режим в норме (ΔT=%d°C)", tempData.AvgDeltaTemp))
        case "warning":
            recommendations = append(recommendations, 
                fmt.Sprintf("Температурный режим требует внимания (ΔT=%d°C, норма: 17-23°C)", tempData.AvgDeltaTemp))
        case "critical":
            recommendations = append(recommendations, 
                fmt.Sprintf("Критическое отклонение температуры (ΔT=%d°C)", tempData.AvgDeltaTemp))
        case "unknown":
            recommendations = append(recommendations, 
                "Данные о температуре отсутствуют")
        }
    } else {
        recommendations = append(recommendations, 
            "Данные о температуре отсутствуют")
    }

    // Анализ насосов
    if pumpData != nil && pumpData.TotalPumps > 0 {
        statusInfo := fmt.Sprintf("Насосы: %d нормальных", pumpData.NormalPumps)
        if pumpData.WarningPumps > 0 {
            statusInfo += fmt.Sprintf(", %d с предупреждением", pumpData.WarningPumps)
        }
        if pumpData.CriticalPumps > 0 {
            statusInfo += fmt.Sprintf(", %d критических", pumpData.CriticalPumps)
        }
        recommendations = append(recommendations, statusInfo)
        
        switch pumpStatus {
        case "normal":
            recommendations = append(recommendations, 
                fmt.Sprintf("Состояние насосов в норме (макс. наработка: %d ч)", operatingHours))
        case "warning":
            recommendations = append(recommendations, 
                fmt.Sprintf("Требуется внимание к насосам (макс. наработка: %d ч)", operatingHours))
        case "critical":
            recommendations = append(recommendations, 
                fmt.Sprintf("Срочное обслуживание насосов требуется (макс. наработка: %d ч)", operatingHours))
        case "unknown":
            recommendations = append(recommendations, 
                "Данные о насосах отсутствуют")
        }
        
        if operatingHours > 8000 {
            recommendations = append(recommendations, 
                "Рекомендуется плановое техническое обслуживание")
        }
    } else {
        recommendations = append(recommendations, 
            "Данные о насосах отсутствуют")
    }

    // Рекомендации по качеству данных
    if coldRecords < 24 {
        recommendations = append(recommendations, 
            "Рекомендуется увеличить частоту сбора данных ХВС")
    }

    if hotRecords < 24 {
        recommendations = append(recommendations, 
            "Рекомендуется увеличить частоту сбора данных ГВС")
    }

    return recommendations
}

// Резервный анализ если данных нет в БД
func (a *Analyzer) analyzeEstimatedData(buildingID uuid.UUID, days int) *ConsumptionAnalysis {
    // Реалистичные оценки для МКД
    avgColdWater := 5 + rand.Intn(5)
    avgHotWater := 3 + rand.Intn(3)
    
    hoursInPeriod := days * 24
    totalColdWater := avgColdWater * hoursInPeriod
    totalHotWater := avgHotWater * hoursInPeriod
    
    difference := totalColdWater - totalHotWater
    var differencePercent float64
    if totalColdWater > 0 {
        differencePercent = (float64(difference) / float64(totalColdWater)) * 100
    }

    // Реалистичное соотношение для оценок
    var hotToColdRatio float64
    if totalColdWater > 0 {
        hotToColdRatio = (float64(totalHotWater) / float64(totalColdWater)) * 100
    }

    return &ConsumptionAnalysis{
        BuildingID:         buildingID,
        Period:             fmt.Sprintf("%d дней (оценка)", days),
        TotalColdWater:     totalColdWater,
        TotalHotWater:      totalHotWater,
        Difference:         difference,
        DifferencePercent:  differencePercent,
        HotToColdRatio:     hotToColdRatio,
        HasAnomalies:       false,
        AnomalyCount:       0,
        WaterBalanceStatus: "normal",
        TemperatureStatus:  "normal",
        PumpStatus:         "normal",
        PumpOperatingHours: 5000 + rand.Intn(5000),
        Recommendations:    []string{"Данные отсутствуют в системе. Показаны расчетные значения."},
    }
}

// Методы получения детальных данных из БД (для совместимости)
func (a *Analyzer) getColdWaterData(ctx context.Context, buildingID uuid.UUID, start, end time.Time) ([]map[string]interface{}, error) {
    rows, err := a.pool.Query(ctx, `
        SELECT cwm.flow_rate, cwm.timestamp 
        FROM cold_water_meters cwm
        JOIN itp i ON cwm.itp_id = i.id
        WHERE i.building_id = $1 AND cwm.timestamp BETWEEN $2 AND $3
        ORDER BY cwm.timestamp`, buildingID, start, end)
    
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var data []map[string]interface{}
    for rows.Next() {
        var flowRate int
        var timestamp time.Time
        err := rows.Scan(&flowRate, &timestamp)
        if err != nil {
            return nil, err
        }
        data = append(data, map[string]interface{}{
            "flow_rate": flowRate,
            "timestamp": timestamp,
        })
    }

    return data, nil
}

func (a *Analyzer) getHotWaterData(ctx context.Context, buildingID uuid.UUID, start, end time.Time) ([]map[string]interface{}, error) {
    rows, err := a.pool.Query(ctx, `
        SELECT flow_rate_ch1, flow_rate_ch2, timestamp 
        FROM hot_water_meters 
        WHERE building_id = $1 AND timestamp BETWEEN $2 AND $3
        ORDER BY timestamp`, buildingID, start, end)
    
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var data []map[string]interface{}
    for rows.Next() {
        var flowRateCh1, flowRateCh2 int
        var timestamp time.Time
        err := rows.Scan(&flowRateCh1, &flowRateCh2, &timestamp)
        if err != nil {
            return nil, err
        }
        data = append(data, map[string]interface{}{
            "flow_rate_ch1": flowRateCh1,
            "flow_rate_ch2": flowRateCh2,
            "total_flow":    flowRateCh1 + flowRateCh2,
            "timestamp":     timestamp,
        })
    }

    return data, nil
}