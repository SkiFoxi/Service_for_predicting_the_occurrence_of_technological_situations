class WaterCharts {
    constructor() {
        this.charts = new Map();
    }

    // Создать график потребления
    createConsumptionChart(canvasId, data) {
        const ctx = document.getElementById(canvasId).getContext('2d');
        
        if (this.charts.has(canvasId)) {
            this.charts.get(canvasId).destroy();
        }

        const chart = new Chart(ctx, {
            type: 'line',
            data: {
                labels: data.labels,
                datasets: [
                    {
                        label: 'ХВС - Подача',
                        data: data.coldWaterIn,
                        borderColor: '#3498db',
                        backgroundColor: 'rgba(52, 152, 219, 0.1)',
                        tension: 0.4
                    },
                    {
                        label: 'ХВС - Возврат',
                        data: data.coldWaterOut,
                        borderColor: '#2980b9',
                        backgroundColor: 'rgba(41, 128, 185, 0.1)',
                        tension: 0.4
                    },
                    {
                        label: 'ГВС - Канал 1',
                        data: data.hotWaterCh1,
                        borderColor: '#e74c3c',
                        backgroundColor: 'rgba(231, 76, 60, 0.1)',
                        tension: 0.4
                    },
                    {
                        label: 'ГВС - Канал 2',
                        data: data.hotWaterCh2,
                        borderColor: '#c0392b',
                        backgroundColor: 'rgba(192, 57, 43, 0.1)',
                        tension: 0.4
                    }
                ]
            },
            options: {
                responsive: true,
                plugins: {
                    title: {
                        display: true,
                        text: 'Потребление воды (м³/ч)'
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true,
                        title: {
                            display: true,
                            text: 'м³/ч'
                        }
                    }
                }
            }
        });

        this.charts.set(canvasId, chart);
        return chart;
    }

    // Создать круговую диаграмму анализа
    createAnalysisChart(canvasId, analysis) {
        const ctx = document.getElementById(canvasId).getContext('2d');
        
        if (this.charts.has(canvasId)) {
            this.charts.get(canvasId).destroy();
        }

        const chart = new Chart(ctx, {
            type: 'doughnut',
            data: {
                labels: ['ХВС', 'ГВС', 'Разница'],
                datasets: [{
                    data: [
                        analysis.totalColdWater,
                        analysis.totalHotWater,
                        analysis.difference
                    ],
                    backgroundColor: [
                        '#3498db',
                        '#e74c3c',
                        '#f39c12'
                    ]
                }]
            },
            options: {
                responsive: true,
                plugins: {
                    legend: {
                        position: 'bottom'
                    },
                    title: {
                        display: true,
                        text: 'Распределение потребления'
                    }
                }
            }
        });

        this.charts.set(canvasId, chart);
        return chart;
    }

    // Обновить данные в реальном времени
    updateRealtimeChart(data) {
        // Логика обновления графика реального времени
    }
}

window.waterCharts = new WaterCharts();