class WaterMonitoringAPI {
    constructor(baseUrl = 'http://localhost:8080/api') {
        this.baseUrl = baseUrl;
    }

    async request(endpoint, options = {}) {
        try {
            const response = await fetch(`${this.baseUrl}${endpoint}`, {
                headers: {
                    'Content-Type': 'application/json',
                    ...options.headers
                },
                ...options
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            return await response.json();
        } catch (error) {
            console.error('API request failed:', error);
            throw error;
        }
    }

    // Получить список всех зданий
    async getBuildings() {
        return this.request('/buildings');
    }

    // Получить анализ для конкретного здания
    async analyzeBuilding(buildingId, days = 30) {
        return this.request(`/analysis/${buildingId}?days=${days}`);
    }

    // Заполнить тестовыми данными
    async seedTestData() {
        return this.request('/seed-data', { method: 'POST' });
    }

    // Получить данные в реальном времени (заглушка)
    async getRealtimeData(buildingId) {
        // В реальном приложении здесь будет WebSocket или частые запросы
        return {
            coldWaterIn: Math.random() * 100,
            coldWaterOut: Math.random() * 80,
            hotWaterCh1: Math.random() * 60,
            hotWaterCh2: Math.random() * 40,
            timestamp: new Date().toISOString()
        };
    }
}

// Создаем глобальный экземпляр API
window.api = new WaterMonitoringAPI();

// web/js/api.js
class WaterMonitoringAPI {
    constructor(baseUrl = 'http://localhost:8080/api') {
        this.baseUrl = baseUrl;
    }

    async request(endpoint, options = {}) {
        try {
            const response = await fetch(`${this.baseUrl}${endpoint}`, {
                headers: {
                    'Content-Type': 'application/json',
                    ...options.headers
                },
                ...options
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            return await response.json();
        } catch (error) {
            console.error('API request failed:', error);
            throw error;
        }
    }

    // Получить список всех зданий
    async getBuildings() {
        return this.request('/buildings');
    }

    // Получить конкретное здание
    async getBuilding(id) {
        return this.request(`/buildings/${id}`);
    }

    // Анализ потребления
    async analyzeBuilding(buildingId, days = 30) {
        return this.request(`/analysis/${buildingId}?days=${days}`);
    }

    // Данные реального времени
    async getRealtimeData(buildingId) {
        return this.request(`/realtime/${buildingId}`);
    }

    // Управление генератором
    async startGenerator() {
        return this.request('/generator/start', { method: 'POST' });
    }

    async stopGenerator() {
        return this.request('/generator/stop', { method: 'POST' });
    }

    async getGeneratorStatus() {
        return this.request('/generator/status');
    }

    // Заполнить тестовыми данными
    async seedTestData() {
        return this.request('/seed-data', { method: 'POST' });
    }
}