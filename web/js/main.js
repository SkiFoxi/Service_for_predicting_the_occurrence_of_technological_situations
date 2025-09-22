// web/js/main.js - ВСЕ В ОДНОМ ФАЙЛЕ

// 1. API класс
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
}

// 2. Главное приложение
class WaterMonitoringApp {
    constructor() {
        this.api = new WaterMonitoringAPI();
        this.buildings = [];
        this.init();
    }

    async init() {
        console.log("🚀 Инициализация приложения...");
        this.setupEventListeners();
        await this.loadBuildings();
        console.log("✅ Приложение готово");
    }

    setupEventListeners() {
        // Навигация
        document.querySelectorAll('.nav-link').forEach(link => {
            link.addEventListener('click', (e) => {
                e.preventDefault();
                this.showSection(link.getAttribute('href').substring(1));
            });
        });

        // Поиск
        document.getElementById('searchInput').addEventListener('input', (e) => {
            this.filterBuildings(e.target.value);
        });

        // Анализ
        document.getElementById('analyzeBtn').addEventListener('click', () => {
            this.analyzeSelectedBuilding();
        });

        // Модальное окно
        document.querySelector('.close').addEventListener('click', () => {
            this.hideModal();
        });

        window.addEventListener('click', (e) => {
            if (e.target === document.getElementById('buildingModal')) {
                this.hideModal();
            }
        });
    }

    async loadBuildings() {
        console.log("🏢 Загрузка зданий...");
        const container = document.getElementById('buildingsList');
        
        try {
            container.innerHTML = '<div class="loading">Загрузка данных...</div>';
            
            const buildings = await this.api.getBuildings();
            console.log("✅ Здания загружены:", buildings);
            
            if (buildings && buildings.length > 0) {
                this.buildings = buildings;
                this.renderBuildings(buildings);
                this.populateBuildingSelect(buildings);
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
            container.innerHTML = '<div class="loading">Нет данных о зданиях</div>';
            return;
        }

        container.innerHTML = buildings.map(building => `
            <div class="building-card" onclick="app.showBuildingDetails('${building.id}')">
                <h3>${building.address}</h3>
                <div class="address">
                    ${building.fias_id ? `ФИАС: ${building.fias_id}` : ''}
                    ${building.unom_id ? ` | УНОМ: ${building.unom_id}` : ''}
                </div>
                <div class="building-meta">
                    <span>Добавлено: ${new Date(building.created_at).toLocaleDateString('ru-RU')}</span>
                </div>
            </div>
        `).join('');
    }

    populateBuildingSelect(buildings) {
        const select = document.getElementById('buildingSelect');
        select.innerHTML = '<option value="">Выберите здание...</option>' +
            buildings.map(b => `<option value="${b.id}">${b.address}</option>`).join('');
    }

    filterBuildings(query) {
        const filtered = this.buildings.filter(building =>
            building.address.toLowerCase().includes(query.toLowerCase())
        );
        this.renderBuildings(filtered);
    }

    showBuildingDetails(buildingId) {
        const building = this.buildings.find(b => b.id === buildingId);
        if (!building) return;

        const modalContent = document.getElementById('modalContent');
        modalContent.innerHTML = `
            <div class="building-info">
                <h3>${building.address}</h3>
                <p><strong>ФИАС:</strong> ${building.fias_id || 'Не указан'}</p>
                <p><strong>УНОМ:</strong> ${building.unom_id || 'Не указан'}</p>
                <p><strong>Создано:</strong> ${new Date(building.created_at).toLocaleString('ru-RU')}</p>
            </div>
            <div class="actions" style="margin-top: 20px;">
                <button class="btn-primary" onclick="app.analyzeBuilding('${building.id}')">Анализировать</button>
            </div>
        `;

        this.showModal();
    }

    async analyzeBuilding(buildingId) {
        try {
            const analysis = await this.api.analyzeBuilding(buildingId);
            this.showAnalysisResults(analysis);
            this.showSection('analysis');
            this.hideModal();
        } catch (error) {
            alert('Ошибка анализа: ' + error.message);
        }
    }

    analyzeSelectedBuilding() {
        const buildingId = document.getElementById('buildingSelect').value;
        if (!buildingId) {
            alert('Выберите здание');
            return;
        }
        this.analyzeBuilding(buildingId);
    }

    showAnalysisResults(analysis) {
        const container = document.getElementById('analysisResults');
        container.innerHTML = `
            <div class="analysis-header">
                <h3>Анализ за период: ${analysis.period}</h3>
                <div class="status ${analysis.has_anomalies ? 'has-anomalies' : 'normal'}">
                    ${analysis.has_anomalies ? '⚠️ Аномалии' : '✅ Норма'}
                </div>
            </div>
            <div class="metrics-grid">
                <div class="metric"><span>ХВС:</span><strong>${analysis.total_cold_water} м³</strong></div>
                <div class="metric"><span>ГВС:</span><strong>${analysis.total_hot_water} м³</strong></div>
                <div class="metric"><span>Разница:</span><strong>${analysis.difference} м³</strong></div>
            </div>
        `;
    }

    showSection(sectionId) {
        document.querySelectorAll('.section').forEach(s => s.classList.remove('active'));
        document.getElementById(sectionId).classList.add('active');
        
        document.querySelectorAll('.nav-link').forEach(link => {
            link.classList.remove('active');
            if (link.getAttribute('href') === `#${sectionId}`) {
                link.classList.add('active');
            }
        });
    }

    showModal() {
        document.getElementById('buildingModal').style.display = 'block';
    }

    hideModal() {
        document.getElementById('buildingModal').style.display = 'none';
    }

    showTestData() {
        console.log("🔄 Используем тестовые данные");
        const testBuildings = [
            {
                id: 'test-1',
                address: 'г. Москва, ул. Ленина, д. 10',
                fias_id: 'fias-001',
                unom_id: 'unom-1001',
                created_at: new Date().toISOString()
            },
            {
                id: 'test-2',
                address: 'г. Москва, пр. Мира, д. 25', 
                fias_id: 'fias-002',
                unom_id: 'unom-1002',
                created_at: new Date().toISOString()
            }
        ];
        
        this.buildings = testBuildings;
        this.renderBuildings(testBuildings);
        this.populateBuildingSelect(testBuildings);
    }
}

// 3. Запуск приложения
console.log("📄 Скрипт загружен, ждем DOM...");
document.addEventListener('DOMContentLoaded', function() {
    console.log("🚀 Запускаем приложение...");
    window.app = new WaterMonitoringApp();
});