class WaterMonitoringApp {
    constructor() {
        this.currentBuilding = null;
        this.buildings = [];
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.loadBuildings();
        this.setupRealtimeUpdates();
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
        try {
            const buildings = await api.getBuildings();
            this.buildings = buildings;
            this.renderBuildings(buildings);
            this.populateBuildingSelect(buildings);
        } catch (error) {
            this.showError('Ошибка загрузки зданий');
        }
    }

    renderBuildings(buildings) {
        const container = document.getElementById('buildingsList');
        
        if (buildings.length === 0) {
            container.innerHTML = '<div class="loading">Нет данных о зданиях</div>';
            return;
        }

        container.innerHTML = buildings.map(building => `
            <div class="building-card" onclick="app.showBuildingDetails('${building.id}')">
                <h3>${building.address}</h3>
                <div class="address">${building.fias_id ? `ФИАС: ${building.fias_id}` : ''}</div>
                <div class="building-meta">
                    <span>УНОМ: ${building.unom_id || 'Не указан'}</span>
                    <span>${new Date(building.created_at).toLocaleDateString()}</span>
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

    async showBuildingDetails(buildingId) {
        const building = this.buildings.find(b => b.id === buildingId);
        if (!building) return;

        const modalContent = document.getElementById('modalContent');
        modalContent.innerHTML = `
            <div class="building-info">
                <h3>${building.address}</h3>
                <p><strong>ФИАС:</strong> ${building.fias_id || 'Не указан'}</p>
                <p><strong>УНОМ:</strong> ${building.unom_id || 'Не указан'}</p>
                <p><strong>Создано:</strong> ${new Date(building.created_at).toLocaleString()}</p>
                <p><strong>Обновлено:</strong> ${new Date(building.updated_at).toLocaleString()}</p>
            </div>
            <div class="actions">
                <button class="btn-primary" onclick="app.analyzeBuilding('${buildingId}')">Анализировать</button>
            </div>
        `;

        this.showModal();
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

    async analyzeBuilding(buildingId, days = 30) {
        try {
            const analysis = await api.analyzeBuilding(buildingId, days);
            this.showAnalysisResults(analysis);
            this.showSection('analysis');
        } catch (error) {
            this.showError('Ошибка анализа данных');
        }
    }

    showAnalysisResults(analysis) {
        const container = document.getElementById('analysisResults');
        
        container.innerHTML = `
            <div class="analysis-header">
                <h3>Анализ за период: ${analysis.period}</h3>
                <div class="status ${analysis.hasAnomalies ? 'has-anomalies' : 'normal'}">
                    ${analysis.hasAnomalies ? 'Обнаружены аномалии' : 'Норма'}
                </div>
            </div>
            
            <div class="metrics-grid">
                <div class="metric">
                    <span>ХВС всего:</span>
                    <strong>${analysis.totalColdWater} м³</strong>
                </div>
                <div class="metric">
                    <span>ГВС всего:</span>
                    <strong>${analysis.totalHotWater} м³</strong>
                </div>
                <div class="metric">
                    <span>Разница:</span>
                    <strong class="${analysis.difference > 0 ? 'positive' : 'negative'}">
                        ${analysis.difference} м³ (${analysis.differencePercent}%)
                    </strong>
                </div>
                <div class="metric">
                    <span>Аномалий:</span>
                    <strong>${analysis.anomalyCount}</strong>
                </div>
            </div>
            
            <div class="chart-container">
                <canvas id="analysisChart" width="400" height="200"></canvas>
            </div>
        `;

        // Создаем график анализа
        waterCharts.createAnalysisChart('analysisChart', analysis);
    }

    showSection(sectionId) {
        // Скрываем все секции
        document.querySelectorAll('.section').forEach(section => {
            section.classList.remove('active');
        });

        // Показываем выбранную секцию
        document.getElementById(sectionId).classList.add('active');

        // Обновляем навигацию
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

    showError(message) {
        alert(`Ошибка: ${message}`);
    }

    setupRealtimeUpdates() {
        // Обновление данных каждые 5 секунд
        setInterval(() => {
            this.updateRealtimeData();
        }, 5000);
    }

    async updateRealtimeData() {
        if (!this.currentBuilding) return;

        try {
            const data = await api.getRealtimeData(this.currentBuilding);
            
            document.getElementById('coldWaterIn').textContent = `${data.coldWaterIn.toFixed(2)} м³/ч`;
            document.getElementById('coldWaterOut').textContent = `${data.coldWaterOut.toFixed(2)} м³/ч`;
            document.getElementById('hotWaterCh1').textContent = `${data.hotWaterCh1.toFixed(2)} м³/ч`;
            document.getElementById('hotWaterCh2').textContent = `${data.hotWaterCh2.toFixed(2)} м³/ч`;

        } catch (error) {
            console.error('Ошибка обновления реального времени:', error);
        }
    }
}

// Инициализация приложения
const app = new WaterMonitoringApp();