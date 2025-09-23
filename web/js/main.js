// web/js/main.js - ПОЛНЫЙ КОД С РЕАЛЬНЫМ ВРЕМЕНЕМ

class WaterMonitoringAPI {
    constructor(baseUrl = 'http://localhost:8080/api') {
        this.baseUrl = baseUrl;
    }

    async request(endpoint, options = {}) {
        const url = `${this.baseUrl}${endpoint}`;
        console.log(`🔄 API запрос: ${url}`);
        
        try {
            const response = await fetch(url, {
                headers: {
                    'Content-Type': 'application/json',
                    ...options.headers
                },
                ...options
            });

            console.log(`📊 Статус ответа: ${response.status}`);
            
            if (!response.ok) {
                throw new Error(`Ошибка HTTP! статус: ${response.status}`);
            }

            const data = await response.json();
            console.log('✅ Данные получены');
            return data;
        } catch (error) {
            console.error('❌ Ошибка API:', error);
            throw error;
        }
    }

    async getBuildings() {
        return this.request('/buildings');
    }

    async analyzeBuilding(buildingId, days = 30) {
        return this.request(`/analysis/${buildingId}?days=${days}`);
    }

    async getRealtimeData(buildingId) {
        return this.request(`/realtime/${buildingId}`);
    }

    async startGenerator() {
        return this.request('/generator/start', { method: 'POST' });
    }

    async stopGenerator() {
        return this.request('/generator/stop', { method: 'POST' });
    }

    async getGeneratorStatus() {
        return this.request('/generator/status');
    }

    async seedTestData() {
        return this.request('/seed-data', { method: 'POST' });
    }
}

class WaterMonitoringApp {
    constructor() {
        this.api = new WaterMonitoringAPI();
        this.buildings = [];
        this.currentBuilding = null;
        this.realtimeInterval = null;
        this.realtimeChart = null;
        this.init();
    }

    async init() {
        console.log("🚀 Инициализация приложения...");
        this.setupEventListeners();
        await this.loadBuildings();
        console.log("✅ Приложение готово");
    }

    setupEventListeners() {
        console.log("🔗 Настройка обработчиков событий...");
        
        // Навигация
        document.querySelectorAll('.nav-link').forEach(link => {
            link.addEventListener('click', (e) => {
                e.preventDefault();
                const section = link.getAttribute('href').substring(1);
                console.log(`📱 Переход в раздел: ${section}`);
                this.showSection(section);
            });
        });

        // Поиск
        const searchInput = document.getElementById('searchInput');
        if (searchInput) {
            searchInput.addEventListener('input', (e) => {
                this.filterBuildings(e.target.value);
            });
        }

        // Анализ
        const analyzeBtn = document.getElementById('analyzeBtn');
        if (analyzeBtn) {
            analyzeBtn.addEventListener('click', () => {
                console.log("📊 Кнопка анализа нажата");
                this.analyzeSelectedBuilding();
            });
        }

        // Модальное окно
        const closeBtn = document.querySelector('.close');
        if (closeBtn) {
            closeBtn.addEventListener('click', () => {
                console.log("❌ Закрытие модального окна");
                this.hideModal();
            });
        }

        window.addEventListener('click', (e) => {
            if (e.target === document.getElementById('buildingModal')) {
                this.hideModal();
            }
        });

        console.log("✅ Обработчики событий настроены");
    }

    async loadBuildings() {
        console.log("🏢 Загрузка зданий...");
        const container = document.getElementById('buildingsList');
        
        try {
            if (container) {
                container.innerHTML = '<div class="loading">Загрузка данных...</div>';
            }
            
            const buildings = await this.api.getBuildings();
            console.log("✅ Здания загружены:", buildings);
            
            if (buildings && buildings.length > 0) {
                this.buildings = buildings;
                this.renderBuildings(buildings);
                this.populateBuildingSelect(buildings);
                
                // Устанавливаем первое здание как текущее для реального времени
                if (buildings.length > 0) {
                    this.currentBuilding = buildings[0].id;
                }
            } else {
                this.showTestData();
            }
            
        } catch (error) {
            console.error("❌ Ошибка загрузки:", error);
            this.showTestData();
        }
    }

    renderBuildings(buildings) {
        const container = document.getElementById('buildingsList');
        if (!buildings || buildings.length === 0) {
            container.innerHTML = '<div class="loading">Объекты не найдены</div>';
            return;
        }

        container.innerHTML = buildings.map(building => `
            <div class="building-row" onclick="app.showBuildingDetails('${building.id}')">
                <div class="building-address">${this.escapeHtml(building.address)}</div>
                <div class="building-id">
                    ФИАС: ${building.fias_id || 'не указан'} | УНОМ: ${building.unom_id || 'не указан'}
                </div>
                <div>Активен</div>
                <div class="building-date">${new Date(building.created_at).toLocaleDateString('ru-RU')}</div>
            </div>
            `).join('');
    }

    populateBuildingSelect(buildings) {
        const select = document.getElementById('buildingSelect');
        if (!select) return;
        
        select.innerHTML = '<option value="">Выберите здание...</option>' +
            buildings.map(b => `<option value="${b.id}">${b.address}</option>`).join('');
        
        console.log(`✅ Выпадающий список заполнен ${buildings.length} зданиями`);
    }

    filterBuildings(query) {
        console.log(`🔍 Фильтрация зданий по запросу: "${query}"`);
        const filtered = this.buildings.filter(building =>
            building.address.toLowerCase().includes(query.toLowerCase())
        );
        this.renderBuildings(filtered);
    }

    showBuildingDetails(buildingId) {
        console.log(`🔍 Показать детали здания: ${buildingId}`);
        const building = this.buildings.find(b => b.id === buildingId);
        if (!building) {
            console.error("❌ Здание не найдено:", buildingId);
            return;
        }

        const modalContent = document.getElementById('modalContent');
        if (!modalContent) return;
        
        modalContent.innerHTML = `
            <div class="building-info">
                <h3>${this.escapeHtml(building.address)}</h3>
                <p><strong>ФИАС:</strong> ${building.fias_id || 'Не указан'}</p>
                <p><strong>УНОМ:</strong> ${building.unom_id || 'Не указан'}</p>
                <p><strong>ID:</strong> ${building.id}</p>
                <p><strong>Создано:</strong> ${new Date(building.created_at).toLocaleString('ru-RU')}</p>
                <p><strong>Обновлено:</strong> ${new Date(building.updated_at).toLocaleString('ru-RU')}</p>
            </div>
            <div class="actions" style="margin-top: 20px;">
                <button class="btn-primary" onclick="app.analyzeBuilding('${building.id}')">Анализировать потребление</button>
                <button class="btn-primary" onclick="app.setRealtimeBuilding('${building.id}')" style="margin-left: 10px;">Мониторить в реальном времени</button>
            </div>
        `;

        this.showModal();
    }

    setRealtimeBuilding(buildingId) {
        console.log(`🎯 Установка здания для мониторинга: ${buildingId}`);
        this.currentBuilding = buildingId;
        this.showSection('realtime');
        this.hideModal();
    }

    async analyzeBuilding(buildingId, days = 30) {
        console.log(`📈 Анализ здания ${buildingId} за ${days} дней`);
        try {
            const analysis = await this.api.analyzeBuilding(buildingId, days);
            this.showAnalysisResults(analysis);
            this.showSection('analysis');
            this.hideModal();
        } catch (error) {
            console.error("❌ Ошибка анализа:", error);
            this.showError('Ошибка анализа данных: ' + error.message);
        }
    }

    async analyzeSelectedBuilding() {
        const buildingId = document.getElementById('buildingSelect').value;
        const days = document.getElementById('periodSelect').value;
        
        if (!buildingId) {
            this.showError('Выберите здание для анализа');
            return;
        }

        await this.analyzeBuilding(buildingId, days);
    }

    // main.js - полный метод showAnalysisResults

showAnalysisResults(analysis) {
    console.log("📊 Отображение результатов анализа:", analysis);
    const container = document.getElementById('analysisResults');
    if (!container) return;

    // Показываем откуда взяты данные
    let htmlContent = '';
    if (analysis.data_source === 'estimated') {
        htmlContent = `
            <div class="warning-banner" style="background: #fff3cd; border: 1px solid #ffeaa7; padding: 15px; border-radius: 4px; margin-bottom: 20px;">
                ⚠️ Внимание: используются расчетные данные. Реальные данные отсутствуют в системе.
            </div>
        `;
    }
    
    // Статусы с иконками
    const statusIcons = {
        normal: '✅',
        leak: '🚨', 
        error: '⚠️',
        warning: '🔶',
        critical: '🔥',
        unknown: '❓'
    };
    
    const pumpIcons = {
        normal: '✅',
        warning: '🔶',
        critical: '🚨',
        maintenance_soon: '⚙️',
        maintenance_required: '🛠️',
        unknown: '❓'
    };

    // Тексты для статусов баланса воды
    const getWaterBalanceText = (status, analysis) => {
        const ratio = analysis.difference_percent ? 
            ` (ГВС/ХВС: ${(100 - analysis.difference_percent).toFixed(1)}%)` : '';
            
        const texts = {
            normal: `Норма${ratio}`,
            leak: `Возможная утечка${ratio}`,
            error: `Ошибка данных${ratio}`,
            warning: `Отклонение от нормы${ratio}`,
            unknown: 'Недостаточно данных'
        };
        return texts[status] || 'Неизвестный статус';
    };

    // Тексты для температурного режима
    const getTemperatureText = (status, analysis) => {
        const tempInfo = analysis.temperature_data ? 
            ` (ΔT=${analysis.temperature_data.avg_delta_temp}°C)` : '';
            
        const texts = {
            normal: `Норма${tempInfo}`,
            warning: `Отклонение${tempInfo}`,
            critical: `Критично${tempInfo}`,
            unknown: 'Данные отсутствуют'
        };
        return texts[status] || 'Неизвестный статус';
    };

    // Тексты для насосов
    const getPumpStatusText = (status, analysis) => {
        const hoursInfo = analysis.pump_operating_hours ? 
            ` (${analysis.pump_operating_hours} ч)` : '';
            
        const texts = {
            normal: `Норма${hoursInfo}`,
            warning: `Требует внимания${hoursInfo}`,
            critical: `Срочное обслуживание${hoursInfo}`,
            maintenance_soon: `ТО в ближайшее время${hoursInfo}`,
            maintenance_required: `СРОЧНОЕ ТЕХОБСЛУЖИВАНИЕ${hoursInfo}`,
            unknown: 'Данные отсутствуют'
        };
        return texts[status] || 'Неизвестный статус';
    };

    htmlContent += `
        <div class="analysis-header">
            <h3>Анализ за период: ${analysis.period || '30 дней'}</h3>
            <div class="status ${analysis.has_anomalies ? 'has-anomalies' : 'normal'}">
                ${analysis.has_anomalies ? '⚠️ Обнаружены аномалии' : '✅ Норма'}
            </div>
        </div>
        
        <div class="metrics-grid">
            <div class="metric">
                <span>ХВС всего:</span>
                <strong>${analysis.total_cold_water || 0} м³</strong>
            </div>
            <div class="metric">
                <span>ГВС всего:</span>
                <strong>${analysis.total_hot_water || 0} м³</strong>
            </div>
            <div class="metric">
                <span>Разница:</span>
                <strong class="${analysis.difference > 0 ? 'positive' : 'negative'}">
                    ${analysis.difference || 0} м³ (${analysis.difference_percent ? analysis.difference_percent.toFixed(1) : 0}%)
                </strong>
            </div>
            <div class="metric">
                <span>Аномалий:</span>
                <strong>${analysis.anomaly_count || 0}</strong>
            </div>
        </div>

        <!-- ИНТЕЛЛЕКТУАЛЬНЫЙ АНАЛИЗ -->
        <div class="intelligent-analysis">
            <h4>🧠 Интеллектуальный анализ</h4>
            
            <div class="analysis-item">
                <strong>Баланс воды:</strong>
                <span class="status-${analysis.water_balance_status}">
                    ${statusIcons[analysis.water_balance_status] || '❓'} 
                    ${getWaterBalanceText(analysis.water_balance_status, analysis)}
                </span>
            </div>
            
            <div class="analysis-item">
                <strong>Температурный режим:</strong>
                <span class="status-${analysis.temperature_status}">
                    ${statusIcons[analysis.temperature_status] || '❓'}
                    ${getTemperatureText(analysis.temperature_status, analysis)}
                </span>
            </div>
            
            <div class="analysis-item">
                <strong>Состояние насосов:</strong>
                <span class="status-${analysis.pump_status}">
                    ${pumpIcons[analysis.pump_status] || '❓'}
                    ${getPumpStatusText(analysis.pump_status, analysis)}
                </span>
            </div>

            <!-- Дополнительная информация если есть детальные данные -->
            ${analysis.temperature_data ? `
            <div class="analysis-item" style="font-size: 12px; color: #666;">
                <strong>Температуры:</strong>
                <span>Подача: ${analysis.temperature_data.avg_supply_temp}°C, 
                Возврат: ${analysis.temperature_data.avg_return_temp}°C, 
                ΔT: ${analysis.temperature_data.avg_delta_temp}°C</span>
            </div>
            ` : ''}

            ${analysis.pump_data ? `
            <div class="analysis-item" style="font-size: 12px; color: #666;">
                <strong>Насосы:</strong>
                <span>Всего: ${analysis.pump_data.total_pumps}, 
                Норма: ${analysis.pump_data.normal_pumps}, 
                Предупреждение: ${analysis.pump_data.warning_pumps}, 
                Критично: ${analysis.pump_data.critical_pumps}</span>
            </div>
            ` : ''}
        </div>

        <!-- РЕКОМЕНДАЦИИ -->
        <div class="recommendations">
            <h4>💡 Рекомендации системы</h4>
            <div class="recommendations-list">
                ${this.renderRecommendations(analysis.recommendations || [])}
            </div>
        </div>
    `;

    container.innerHTML = htmlContent;
}

// Вспомогательный метод для рендеринга рекомендаций
renderRecommendations(recommendations) {
    if (!recommendations || recommendations.length === 0) {
        return '<div class="recommendation">✅ Все системы работают нормально</div>';
    }
    
    return recommendations.map(rec => {
        // Определяем иконку по содержанию рекомендации
        let icon = '💡';
        if (rec.includes('🚨') || rec.includes('ВНИМАНИЕ') || rec.includes('СРОЧНО')) {
            icon = '🚨';
        } else if (rec.includes('⚠️') || rec.includes('Внимание') || rec.includes('Требуется')) {
            icon = '⚠️';
        } else if (rec.includes('✅') || rec.includes('Норма') || rec.includes('норме')) {
            icon = '✅';
        } else if (rec.includes('🔶') || rec.includes('Отклонение') || rec.includes('наблюдение')) {
            icon = '🔶';
        } else if (rec.includes('⚙️') || rec.includes('ТО') || rec.includes('обслуживание')) {
            icon = '⚙️';
        } else if (rec.includes('📊') || rec.includes('данн')) {
            icon = '📊';
        }
        
        return `<div class="recommendation">${icon} ${this.escapeHtml(rec)}</div>`;
    }).join('');
}

// Вспомогательные методы для текста статусов
    getWaterBalanceText(status) {
        const texts = {
            normal: 'Норма (подача ≈ возврат + потребление)',
            leak: 'Возможная утечка (нарушен баланс)',
            error: 'Ошибка баланса (большое отклонение)'
        };
        return texts[status] || 'Неизвестный статус';
    }

    getTemperatureText(status) {
        const texts = {
            normal: 'Норма (ΔT = 17-23°C)',
            warning: 'Предупреждение (ΔT вне нормы)',
            critical: 'Критично (ΔT критическое)'
        };
        return texts[status] || 'Неизвестный статус';
    }

    getPumpStatusText(status) {
        const texts = {
            normal: 'Норма',
            maintenance_soon: 'Требуется ТО в ближайшее время',
            maintenance_required: 'СРОЧНОЕ ТЕХОБСЛУЖИВАНИЕ!'
        };
        return texts[status] || 'Неизвестный статус';
    }

    renderRecommendations(recommendations) {
        if (!recommendations || recommendations.length === 0) {
            return '<div class="recommendation">✅ Все системы работают нормально</div>';
        }
        
        return recommendations.map(rec => 
            `<div class="recommendation">${rec}</div>`
        ).join('');
    }

    showSection(sectionId) {
        console.log(`📱 Переключение на раздел: ${sectionId}`);
        
        // Останавливаем мониторинг реального времени при переключении секций
        this.stopRealtimeMonitoring();
        
        // Скрываем все секции
        document.querySelectorAll('.section').forEach(section => {
            section.classList.remove('active');
        });

        // Показываем выбранную секцию
        const activeSection = document.getElementById(sectionId);
        if (activeSection) {
            activeSection.classList.add('active');
        }

        // Обновляем навигацию
        document.querySelectorAll('.nav-link').forEach(link => {
            link.classList.remove('active');
            if (link.getAttribute('href') === `#${sectionId}`) {
                link.classList.add('active');
            }
        });
        
        // Запускаем мониторинг если перешли в секцию реального времени
        if (sectionId === 'realtime') {
            setTimeout(() => {
                this.startRealtimeMonitoring();
            }, 100);
        }
    }

    // МОНИТОРИНГ В РЕАЛЬНОМ ВРЕМЕНИ
    startRealtimeMonitoring() {
        console.log("⏰ Запуск мониторинга в реальном времени");
        
        // Останавливаем предыдущий интервал если есть
        this.stopRealtimeMonitoring();
        
        // Обновляем данные каждые 3 секунды
        this.realtimeInterval = setInterval(() => {
            this.updateRealtimeData();
        }, 3000);
        
        // Первое обновление сразу
        this.updateRealtimeData();
    }

    stopRealtimeMonitoring() {
        if (this.realtimeInterval) {
            clearInterval(this.realtimeInterval);
            this.realtimeInterval = null;
            console.log("⏹️ Остановлен мониторинг реального времени");
        }
    }

    async updateRealtimeData() {
        if (!this.currentBuilding && this.buildings.length > 0) {
            this.currentBuilding = this.buildings[0].id;
        }

        if (!this.currentBuilding) {
            console.log("⚠️ Нет здания для мониторинга");
            this.displayDemoRealtimeData();
            return;
        }

        try {
            const data = await this.api.getRealtimeData(this.currentBuilding);
            this.displayRealtimeData(data);
        } catch (error) {
            console.error("❌ Ошибка обновления реального времени:", error);
            // Используем демо-данные если API недоступно
            this.displayDemoRealtimeData();
        }
    }

    displayRealtimeData(data) {
        console.log("📊 Обновление данных реального времени:", data);
        
        // Холодная вода - подача
        const coldWaterIn = data.cold_water?.total_flow_rate || data.coldWaterIn || 0;
        document.getElementById('coldWaterIn').textContent = `${coldWaterIn} м³/ч`;
        
        // Горячая вода - канал 1
        const hotWaterCh1 = data.hot_water?.flow_rate_ch1 || data.hotWaterCh1 || 0;
        document.getElementById('hotWaterCh1').textContent = `${hotWaterCh1} м³/ч`;
        
        // Горячая вода - канал 2  
        const hotWaterCh2 = data.hot_water?.flow_rate_ch2 || data.hotWaterCh2 || 0;
        document.getElementById('hotWaterCh2').textContent = `${hotWaterCh2} м³/ч`;
        
        // Холодная вода - возврат (примерно 80% от подачи)
        const coldWaterOut = Math.round(coldWaterIn * 0.8);
        document.getElementById('coldWaterOut').textContent = `${coldWaterOut} м³/ч`;
        
        // Обновляем тренды
        this.updateTrends();
        
        // Обновляем график
        this.updateRealtimeChart(data);
    }

    displayDemoRealtimeData() {
        console.log("🎮 Использование демо-данных реального времени");
        
        // Демо-данные когда API недоступно
        const coldWaterIn = 50 + Math.floor(Math.random() * 50); // 50-100
        const hotWaterCh1 = 20 + Math.floor(Math.random() * 30); // 20-50
        const hotWaterCh2 = 10 + Math.floor(Math.random() * 20); // 10-30
        const coldWaterOut = Math.round(coldWaterIn * 0.8); // 80% от подачи
    
        document.getElementById('coldWaterIn').textContent = `${coldWaterIn} м³/ч`;
        document.getElementById('coldWaterOut').textContent = `${coldWaterOut} м³/ч`;
        document.getElementById('hotWaterCh1').textContent = `${hotWaterCh1} м³/ч`;
        document.getElementById('hotWaterCh2').textContent = `${hotWaterCh2} м³/ч`;
        
        this.updateTrends();
        this.updateDemoChart(coldWaterIn, coldWaterOut, hotWaterCh1, hotWaterCh2);
    }

    updateTrends() {
        // Случайные тренды для демонстрации
        const trends = [
            { class: 'up', text: '+' + (1 + Math.floor(Math.random() * 10)) + '%' },
            { class: 'down', text: '-' + (1 + Math.floor(Math.random() * 5)) + '%' },
            { class: 'stable', text: '0%' }
        ];
        
        const trendElements = document.querySelectorAll('.trend');
        trendElements.forEach((element, index) => {
            const trend = trends[Math.floor(Math.random() * trends.length)];
            element.className = 'trend ' + trend.class;
            element.textContent = trend.text;
        });
    }

    updateRealtimeChart(data) {
        const ctx = document.getElementById('realtimeChart');
        if (!ctx) return;
        
        const coldWaterIn = data.cold_water?.total_flow_rate || 0;
        const coldWaterOut = Math.round(coldWaterIn * 0.8);
        const hotWaterCh1 = data.hot_water?.flow_rate_ch1 || 0;
        const hotWaterCh2 = data.hot_water?.flow_rate_ch2 || 0;
        
        this.createOrUpdateChart(ctx, coldWaterIn, coldWaterOut, hotWaterCh1, hotWaterCh2);
    }

    updateDemoChart(coldWaterIn, coldWaterOut, hotWaterCh1, hotWaterCh2) {
        const ctx = document.getElementById('realtimeChart');
        if (!ctx) return;
        
        this.createOrUpdateChart(ctx, coldWaterIn, coldWaterOut, hotWaterCh1, hotWaterCh2);
    }

    createOrUpdateChart(ctx, coldWaterIn, coldWaterOut, hotWaterCh1, hotWaterCh2) {
        if (!this.realtimeChart) {
            this.realtimeChart = new Chart(ctx, {
                type: 'bar',
                data: {
                    labels: ['ХВС подача', 'ХВС возврат', 'ГВС канал 1', 'ГВС канал 2'],
                    datasets: [{
                        label: 'Расход воды (м³/ч)',
                        data: [coldWaterIn, coldWaterOut, hotWaterCh1, hotWaterCh2],
                        backgroundColor: [
                            'rgba(52, 152, 219, 0.7)',
                            'rgba(41, 128, 185, 0.7)',
                            'rgba(231, 76, 60, 0.7)',
                            'rgba(192, 57, 43, 0.7)'
                        ],
                        borderColor: [
                            '#3498db',
                            '#2980b9',
                            '#e74c3c',
                            '#c0392b'
                        ],
                        borderWidth: 1
                    }]
                },
                options: {
                    responsive: true,
                    plugins: {
                        title: {
                            display: true,
                            text: 'Текущий расход воды'
                        }
                    },
                    scales: {
                        y: {
                            beginAtZero: true,
                            title: {
                                display: true,
                                text: 'м³/час'
                            }
                        }
                    }
                }
            });
        } else {
            // Обновляем данные графика
            this.realtimeChart.data.datasets[0].data = [coldWaterIn, coldWaterOut, hotWaterCh1, hotWaterCh2];
            this.realtimeChart.update('active');
        }
    }

    showModal() {
        const modal = document.getElementById('buildingModal');
        if (modal) {
            modal.style.display = 'block';
        }
    }

    hideModal() {
        const modal = document.getElementById('buildingModal');
        if (modal) {
            modal.style.display = 'none';
        }
    }

    showError(message) {
        console.error("❌ Показать ошибку:", message);
        alert(`Ошибка: ${message}`);
    }

    showTestData() {
        console.log("🔄 Используем тестовые данные");
        const testBuildings = [
            {
                id: 'test-1',
                address: 'г. Москва, ул. Ленина, д. 10',
                fias_id: 'fias-001',
                unom_id: 'unom-1001',
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            },
            {
                id: 'test-2',
                address: 'г. Москва, пр. Мира, д. 25', 
                fias_id: 'fias-002',
                unom_id: 'unom-1002',
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            },
            {
                id: 'test-3',
                address: 'г. Москва, ул. Гагарина, д. 15',
                fias_id: 'fias-003',
                unom_id: 'unom-1003',
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            }
        ];
        
        this.buildings = testBuildings;
        this.renderBuildings(testBuildings);
        this.populateBuildingSelect(testBuildings);
        
        // Устанавливаем первое здание для мониторинга
        if (testBuildings.length > 0) {
            this.currentBuilding = testBuildings[0].id;
        }
    }

    escapeHtml(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

// Запуск приложения после загрузки DOM
console.log("📄 Скрипт main.js загружен");
document.addEventListener('DOMContentLoaded', function() {
    console.log("🚀 DOM готов, запускаем приложение...");
    window.app = new WaterMonitoringApp();
});