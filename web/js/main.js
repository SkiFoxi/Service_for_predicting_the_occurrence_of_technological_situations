// web/js/main.js - ПОЛНЫЙ КОД С WebSocket И РЕАЛЬНЫМ ВРЕМЕНЕМ

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

    async createTestBuildings() {
        return this.request('/create-test-buildings', { method: 'POST' });
    }

    async generateCompleteHistory(days = 7) {
        return this.request(`/generate-complete-history?days=${days}`, { method: 'POST' });
    }
}

class RealtimeManager {
    constructor() {
        this.ws = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 3000;
        this.messageHandlers = new Map();
        this.isConnected = false;
    }

    connect() {
        try {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}/ws`;
            
            this.ws = new WebSocket(wsUrl);
            
            this.ws.onopen = () => {
                console.log('🔗 WebSocket connected');
                this.reconnectAttempts = 0;
                this.isConnected = true;
                
                // Запрашиваем подписку на обновления
                this.ws.send(JSON.stringify({
                    type: 'subscribe',
                    channels: ['realtime_updates']
                }));

                // Оповещаем обработчики о подключении
                this.notifyHandlers('connected', {});
            };

            this.ws.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    console.log('📨 WebSocket message:', data.type);
                    this.handleMessage(data);
                } catch (error) {
                    console.error('Error parsing WebSocket message:', error);
                }
            };

            this.ws.onclose = (event) => {
                console.log('🔌 WebSocket disconnected:', event.code, event.reason);
                this.isConnected = false;
                this.notifyHandlers('disconnected', { code: event.code, reason: event.reason });
                this.handleReconnect();
            };

            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
                this.isConnected = false;
                this.notifyHandlers('error', { error });
            };

        } catch (error) {
            console.error('WebSocket connection failed:', error);
            this.handleReconnect();
        }
    }

    handleMessage(data) {
        this.notifyHandlers(data.type, data);
    }

    notifyHandlers(type, data) {
        if (this.messageHandlers.has(type)) {
            this.messageHandlers.get(type).forEach(handler => {
                try {
                    handler(data);
                } catch (error) {
                    console.error('Error in message handler:', error);
                }
            });
        }
    }

    on(messageType, handler) {
        if (!this.messageHandlers.has(messageType)) {
            this.messageHandlers.set(messageType, []);
        }
        this.messageHandlers.get(messageType).push(handler);
    }

    off(messageType, handler) {
        if (this.messageHandlers.has(messageType)) {
            const handlers = this.messageHandlers.get(messageType);
            const index = handlers.indexOf(handler);
            if (index > -1) {
                handlers.splice(index, 1);
            }
        }
    }

    send(message) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(message));
        } else {
            console.warn('WebSocket not connected, message not sent:', message);
        }
    }

    handleReconnect() {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            const delay = this.reconnectDelay * this.reconnectAttempts;
            console.log(`🔄 Attempting to reconnect in ${delay}ms... (${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
            
            setTimeout(() => {
                this.connect();
            }, delay);
        } else {
            console.error('❌ Max reconnection attempts reached');
            this.notifyHandlers('reconnect_failed', {});
        }
    }

    disconnect() {
        if (this.ws) {
            this.ws.close(1000, 'Manual disconnect');
            this.ws = null;
        }
        this.isConnected = false;
    }

    getConnectionStatus() {
        return this.isConnected;
    }
}

class WaterMonitoringApp {
    constructor() {
        this.api = new WaterMonitoringAPI();
        this.realtimeManager = new RealtimeManager();
        this.buildings = [];
        this.currentBuilding = null;
        this.realtimeChart = null;
        this.lastUpdateId = null;
        this.isRealtimeActive = false;
        this.init();
    }

    async init() {
        console.log("🚀 Инициализация приложения...");
        this.setupEventListeners();
        this.setupRealtimeHandlers();
        await this.loadBuildings();
        
        // Подключаемся к WebSocket
        this.realtimeManager.connect();
        
        console.log("✅ Приложение готово");
    }

    setupRealtimeHandlers() {
        // Обработчик подключения WebSocket
        this.realtimeManager.on('connected', (data) => {
            this.showNotification('🔗 Подключено к серверу в реальном времени', 'success');
            this.updateConnectionStatus(true);
        });

        // Обработчик отключения WebSocket
        this.realtimeManager.on('disconnected', (data) => {
            this.showNotification('🔌 Отключено от сервера', 'warning');
            this.updateConnectionStatus(false);
        });

        // Обработчик обновлений реального времени
        this.realtimeManager.on('realtime_update', (data) => {
            if (this.isRealtimeActive && data.building_id === this.currentBuilding) {
                this.handleRealtimeUpdate(data);
            }
        });

        // Обработчик ошибок
        this.realtimeManager.on('error', (data) => {
            console.error('WebSocket error:', data.error);
            this.showNotification('❌ Ошибка соединения', 'error');
        });

        // Пинг-понг для поддержания соединения
        setInterval(() => {
            if (this.realtimeManager.getConnectionStatus()) {
                this.realtimeManager.send({ type: 'ping' });
            }
        }, 30000); // Каждые 30 секунд
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

        // Генератор данных
        const startGeneratorBtn = document.getElementById('startGeneratorBtn');
        if (startGeneratorBtn) {
            startGeneratorBtn.addEventListener('click', () => {
                this.startDataGenerator();
            });
        }

        const stopGeneratorBtn = document.getElementById('stopGeneratorBtn');
        if (stopGeneratorBtn) {
            stopGeneratorBtn.addEventListener('click', () => {
                this.stopDataGenerator();
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
                <button class="btn-primary" onclick="app.setRealtimeBuilding('${building.id}')" style="margin-left: 10px;">Мониторинг в реальном времени</button>
                <button class="btn-primary" onclick="app.generateBuildingData('${building.id}')" style="margin-left: 10px; background: #27ae60;">Сгенерировать данные</button>
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

    showAnalysisResults(analysis) {
        console.log("📊 Отображение результатов анализа:", analysis);
        const container = document.getElementById('analysisResults');
        if (!container) return;

        let htmlContent = '';
        if (analysis.data_source === 'estimated') {
            htmlContent = `
                <div class="warning-banner" style="background: #fff3cd; border: 1px solid #ffeaa7; padding: 15px; border-radius: 4px; margin-bottom: 20px; color: #856404;">
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
                    <span>Соотношение ГВС/ХВС:</span>
                    <strong>${analysis.hot_to_cold_ratio ? analysis.hot_to_cold_ratio.toFixed(1) + '%' : 'N/A'}</strong>
                </div>
            </div>

            <!-- ИНТЕЛЛЕКТУАЛЬНЫЙ АНАЛИЗ -->
            <div class="intelligent-analysis">
                <h4>🧠 Интеллектуальный анализ</h4>
                
                <div class="analysis-item">
                    <strong>Баланс воды:</strong>
                    <span class="status-${analysis.water_balance_status}">
                        ${statusIcons[analysis.water_balance_status] || '❓'} 
                        ${this.getWaterBalanceText(analysis.water_balance_status, analysis)}
                    </span>
                </div>
                
                <div class="analysis-item">
                    <strong>Температурный режим:</strong>
                    <span class="status-${analysis.temperature_status}">
                        ${statusIcons[analysis.temperature_status] || '❓'}
                        ${this.getTemperatureText(analysis.temperature_status, analysis)}
                    </span>
                </div>
                
                <div class="analysis-item">
                    <strong>Состояние насосов:</strong>
                    <span class="status-${analysis.pump_status}">
                        ${pumpIcons[analysis.pump_status] || '❓'}
                        ${this.getPumpStatusText(analysis.pump_status, analysis)}
                    </span>
                </div>
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

    // Вспомогательные методы для текста статусов
    getWaterBalanceText(status, analysis) {
        const ratio = analysis.hot_to_cold_ratio ? ` (${analysis.hot_to_cold_ratio.toFixed(1)}%)` : '';
        const texts = {
            normal: `Норма${ratio}`,
            leak: `Возможная утечка${ratio}`,
            error: `Ошибка данных${ratio}`,
            warning: `Отклонение от нормы${ratio}`,
            unknown: 'Недостаточно данных'
        };
        return texts[status] || 'Неизвестный статус';
    }

    getTemperatureText(status, analysis) {
        const tempInfo = analysis.temperature_data ? 
            ` (ΔT=${analysis.temperature_data.avg_delta_temp}°C)` : '';
        const texts = {
            normal: `Норма${tempInfo}`,
            warning: `Отклонение${tempInfo}`,
            critical: `Критично${tempInfo}`,
            unknown: 'Данные отсутствуют'
        };
        return texts[status] || 'Неизвестный статус';
    }

    getPumpStatusText(status, analysis) {
        const hoursInfo = analysis.pump_operating_hours ? 
            ` (${analysis.pump_operating_hours} ч)` : '';
        const texts = {
            normal: `Норма${hoursInfo}`,
            warning: `Требует внимания${hoursInfo}`,
            critical: `Срочное обслуживание${hoursInfo}`,
            unknown: 'Данные отсутствуют'
        };
        return texts[status] || 'Неизвестный статус';
    }

    renderRecommendations(recommendations) {
        if (!recommendations || recommendations.length === 0) {
            return '<div class="recommendation">✅ Все системы работают нормально</div>';
        }
        
        return recommendations.map(rec => {
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

    showSection(sectionId) {
        console.log(`📱 Переключение на раздел: ${sectionId}`);
        
        // Управление режимом реального времени
        if (sectionId === 'realtime') {
            this.startRealtimeMonitoring();
        } else {
            this.stopRealtimeMonitoring();
        }
        
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
    }

    // РЕАЛЬНОЕ ВРЕМЯ С WebSocket
    startRealtimeMonitoring() {
        console.log("⏰ Запуск мониторинга в реальном времени");
        this.isRealtimeActive = true;
        
        // Загружаем начальные данные
        this.updateRealtimeData();
        
        this.showNotification('🔴 Режим реального времени активирован', 'success');
    }

    stopRealtimeMonitoring() {
        console.log("⏹️ Остановка мониторинга реального времени");
        this.isRealtimeActive = false;
    }

    handleRealtimeUpdate(data) {
        console.log("📊 Обновление данных реального времени:", data);
        this.displayRealtimeData(data.data);
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
            this.displayDemoRealtimeData();
        }
    }

    displayRealtimeData(data) {
        // Холодная вода - подача
        const coldWaterIn = data.cold_water?.total_flow_rate || data.coldWaterIn || 0;
        this.updateMetric('coldWaterIn', coldWaterIn, 'м³/ч');
        
        // Горячая вода - канал 1
        const hotWaterCh1 = data.hot_water?.flow_rate_ch1 || data.hotWaterCh1 || 0;
        this.updateMetric('hotWaterCh1', hotWaterCh1, 'м³/ч');
        
        // Горячая вода - канал 2  
        const hotWaterCh2 = data.hot_water?.flow_rate_ch2 || data.hotWaterCh2 || 0;
        this.updateMetric('hotWaterCh2', hotWaterCh2, 'м³/ч');
        
        // Холодная вода - возврат (примерно 80% от подачи)
        const coldWaterOut = Math.round(coldWaterIn * 0.8);
        this.updateMetric('coldWaterOut', coldWaterOut, 'м³/ч');
        
        // Температурные данные
        if (data.temperature) {
            this.updateTemperatureData(data.temperature);
        }
        
        // Обновляем тренды
        this.updateTrends();
        
        // Обновляем график
        this.updateRealtimeChart(coldWaterIn, coldWaterOut, hotWaterCh1, hotWaterCh2);
        
        // Обновляем timestamp
        const timestampElement = document.getElementById('realtimeTimestamp');
        if (timestampElement) {
            timestampElement.textContent = new Date().toLocaleTimeString('ru-RU');
        }
    }

    updateMetric(elementId, value, unit) {
        const element = document.getElementById(elementId);
        if (element) {
            element.textContent = `${value} ${unit}`;
        }
    }

    updateTemperatureData(tempData) {
        const tempElement = document.getElementById('temperatureData');
        if (tempElement && tempData.supply_temp) {
            tempElement.innerHTML = `
                <div style="display: flex; justify-content: space-around; margin-top: 10px; font-size: 12px; color: #666;">
                    <span>Подача: ${tempData.supply_temp}°C</span>
                    <span>Возврат: ${tempData.return_temp}°C</span>
                    <span>ΔT: ${tempData.delta_temp}°C</span>
                </div>
            `;
        }
    }

    displayDemoRealtimeData() {
        console.log("🎮 Использование демо-данных реального времени");
        
        // Демо-данные когда API недоступно
        const coldWaterIn = 50 + Math.floor(Math.random() * 50);
        const hotWaterCh1 = 20 + Math.floor(Math.random() * 30);
        const hotWaterCh2 = 10 + Math.floor(Math.random() * 20);
        const coldWaterOut = Math.round(coldWaterIn * 0.8);
    
        this.updateMetric('coldWaterIn', coldWaterIn, 'м³/ч');
        this.updateMetric('coldWaterOut', coldWaterOut, 'м³/ч');
        this.updateMetric('hotWaterCh1', hotWaterCh1, 'м³/ч');
        this.updateMetric('hotWaterCh2', hotWaterCh2, 'м³/ч');
        
        this.updateTrends();
        this.updateDemoChart(coldWaterIn, coldWaterOut, hotWaterCh1, hotWaterCh2);
    }

    updateTrends() {
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

    updateRealtimeChart(coldWaterIn, coldWaterOut, hotWaterCh1, hotWaterCh2) {
        const ctx = document.getElementById('realtimeChart');
        if (!ctx) return;
        
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
            this.realtimeChart.data.datasets[0].data = [coldWaterIn, coldWaterOut, hotWaterCh1, hotWaterCh2];
            this.realtimeChart.update('active');
        }
    }

    // Управление генератором данных
    async startDataGenerator() {
        try {
            await this.api.startGenerator();
            this.showNotification('🚀 Генератор данных запущен', 'success');
        } catch (error) {
            this.showError('Ошибка запуска генератора: ' + error.message);
        }
    }

    async stopDataGenerator() {
        try {
            await this.api.stopGenerator();
            this.showNotification('🛑 Генератор данных остановлен', 'warning');
        } catch (error) {
            this.showError('Ошибка остановки генератора: ' + error.message);
        }
    }

    async generateBuildingData(buildingId) {
        try {
            await this.api.generateCompleteHistory(1); // 1 день данных
            this.showNotification('📊 Данные успешно сгенерированы', 'success');
            this.hideModal();
        } catch (error) {
            this.showError('Ошибка генерации данных: ' + error.message);
        }
    }

    // Вспомогательные методы
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
        this.showNotification(message, 'error');
    }

    showNotification(message, type = 'info') {
        // Создаем уведомление
        const notification = document.createElement('div');
        notification.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            padding: 15px 20px;
            border-radius: 4px;
            color: white;
            z-index: 10000;
            max-width: 400px;
            font-size: 14px;
            transition: all 0.3s ease;
        `;
        
        const bgColors = {
            success: '#27ae60',
            error: '#e74c3c',
            warning: '#f39c12',
            info: '#3498db'
        };
        
        notification.style.background = bgColors[type] || bgColors.info;
        notification.textContent = message;
        
        document.body.appendChild(notification);
        
        // Автоматически удаляем через 5 секунд
        setTimeout(() => {
            notification.style.opacity = '0';
            setTimeout(() => {
                if (notification.parentNode) {
                    notification.parentNode.removeChild(notification);
                }
            }, 300);
        }, 5000);
    }

    updateConnectionStatus(connected) {
        const statusElement = document.getElementById('connectionStatus');
        if (statusElement) {
            statusElement.textContent = connected ? '🟢 Подключено' : '🔴 Отключено';
            statusElement.style.color = connected ? '#27ae60' : '#e74c3c';
        }
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
            }
        ];
        
        this.buildings = testBuildings;
        this.renderBuildings(testBuildings);
        this.populateBuildingSelect(testBuildings);
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