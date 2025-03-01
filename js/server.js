const http = require('http');
const os = require('os');
const fs = require('fs');

// Configuración para Raspberry Pi
const CONFIG = {
    tty: '/dev/tty1',        // Forzar salida a TTY1
    updateInterval: 50,
    clearCommand: '\x1b[2J\x1b[H\x1b[?25l',  // También oculta el cursor
    resetCommand: '\x1b[?25h\x1b[0m',        // Restaura el cursor y formato
    centerScreen: true,
    defaultRows: 30,         // Altura por defecto de la terminal
    defaultColumns: 100,     // Ancho por defecto de la terminal
    traceInterval: 3000,     // Intervalo para apariciones
    traceDuration: 4000,     // Duración de la transición
    maxTraceSquares: 3,      // Límite máximo de cuadros activos
    targetFps: 5             // FPS objetivo base
};

// Grid colors completo del logo MDS (16x16)
const GRID_COLORS = [
    "#111111", "#111111", "#111111", "#111111", "#111111", "#111111", "#111111", "#111111", "#111111", "#111111", "#111111", "#111111", "#111111", "#111111", "#111111", "#111111",
    "#111111", "#111111", "#111111", "#111111", "#111111", "#111111", "#111111", "#111111", "#262626", "#404040", "#1d1d1d", "#111111", "#111111", "#111111", "#111111", "#111111",
    "#111111", "#111111", "#111111", "#111111", "#111111", "#111111", "#262626", "#0f0f0f", "#404040", "#171717", "#2b2b2b", "#3e3e3e", "#111111", "#111111", "#111111", "#111111",
    "#111111", "#111111", "#111111", "#111111", "#111111", "#171717", "#0f0f0f", "#101010", "#101010", "#111111", "#111111", "#404040", "#111111", "#111111", "#111111", "#111111",
    "#111111", "#111111", "#111111", "#111111", "#151515", "#3f3f3f", "#111111", "#101010", "#111111", "#111111", "#1f1f1f", "#c7c7c7", "#111111", "#111111", "#111111", "#111111",
    "#111111", "#111111", "#111111", "#111111", "#272727", "#111111", "#111111", "#0f0f0f", "#111111", "#111111", "#1f1f1f", "#767676", "#111111", "#111111", "#111111", "#111111",
    "#111111", "#111111", "#111111", "#111111", "#1d1d1d", "#929292", "#404040", "#262626", "#111111", "#111111", "#121212", "#767676", "#1b1b1b", "#111111", "#111111", "#111111",
    "#111111", "#111111", "#111111", "#111111", "#0f0f0f", "#929292", "#404040", "#3a3a3a", "#111111", "#111111", "#1d1d1d", "#bbbbbb", "#111111", "#111111", "#111111", "#111111",
    "#111111", "#111111", "#111111", "#111111", "#111111", "#1b1b1b", "#5b5b5b", "#232323", "#111111", "#111111", "#111111", "#3c3c3c", "#111111", "#111111", "#111111", "#111111",
    "#111111", "#111111", "#111111", "#111111", "#111111", "#5e5e5e", "#5e5e5e", "#5e5e5e", "#0f0f0f", "#111111", "#262626", "#9b9b9b", "#111111", "#111111", "#111111", "#111111",
    "#111111", "#111111", "#111111", "#111111", "#111111", "#5e5e5e", "#111111", "#1a1a1a", "#0f0f0f", "#111111", "#1b1b1b", "#111111", "#111111", "#111111", "#111111", "#111111",
    "#111111", "#111111", "#111111", "#111111", "#3e3e3e", "#5e5e5e", "#111111", "#111111", "#1b1b1b", "#515151", "#111111", "#111111", "#111111", "#111111", "#111111", "#111111",
    "#111111", "#111111", "#171717", "#171717", "#5e5e5e", "#242424", "#111111", "#111111", "#1b1b1b", "#a2a2a2", "#262626", "#111111", "#111111", "#111111", "#111111", "#111111",
    "#171717", "#171717", "#111111", "#111111", "#111111", "#131313", "#111111", "#111111", "#5d5d5d", "#1a1a1a", "#3a3a3a", "#262626", "#111111", "#111111", "#111111", "#111111",
    "#171717", "#111111", "#111111", "#111111", "#111111", "#111111", "#222222", "#111111", "#1a1a1a", "#1a1a1a", "#1a1a1a", "#2b2b2b", "#222222", "#3a3a3a", "#111111", "#111111",
    "#111111", "#111111", "#111111", "#111111", "#111111", "#111111", "#111111", "#171717", "#171717", "#171717", "#2b2b2b", "#2b2b2b", "#222222", "#222222", "#3a3a3a", "#111111"
];

class MDSRenderer {
    constructor() {
        this.width = 16;
        this.height = 16;
        this.traceSquares = new Set();
        this.currentPositions = Array.from({ length: this.width * this.height }, (_, i) => i);
        this.isAnimating = false;
        this.selectedAdjacent = new Set();
        this.lastTraceUpdate = Date.now();
        this.TRACE_INTERVAL = CONFIG.traceInterval;
        this.TRACE_DURATION = CONFIG.traceDuration;
        this.fadeSteps = ['█', '▓', '▒', '░', ' ', ' '];
        this.maxTraceSquares = CONFIG.maxTraceSquares;
        this.lastCpuUsage = this.getCpuUsage();
        this.ttyStream = null;
        
        // Inicializar stream para TTY
        try {
            // Verificar permisos antes de abrir
            try {
                fs.accessSync(CONFIG.tty, fs.constants.W_OK);
                this.ttyStream = fs.createWriteStream(CONFIG.tty, { flags: 'w' });
                this.ttyInitialized = true;
            } catch (err) {
                console.warn(`No se tiene acceso a ${CONFIG.tty}, usando stdout como fallback`);
                this.ttyInitialized = false;
            }
            
            // Configurar terminal
            this.writeToOutput(CONFIG.clearCommand);
            
            // Forzar dimensiones de terminal si no se pueden detectar
            this.terminalRows = CONFIG.defaultRows;
            this.terminalColumns = CONFIG.defaultColumns;
        } catch (error) {
            console.error('Error configurando salida:', error);
            this.ttyInitialized = false;
        }
    }

    // Método para obtener carga de CPU (cacheado para mejorar rendimiento)
    getCpuUsage() {
        if (!this.lastCpuCheck || Date.now() - this.lastCpuCheck > 500) {
            this.lastCpuCheck = Date.now();
            this.cachedCpuLoad = os.loadavg()[0];
        }
        return this.cachedCpuLoad;
    }

    writeToOutput(data) {
        try {
            if (this.ttyInitialized && this.ttyStream) {
                this.ttyStream.write(data);
            } else {
                process.stdout.write(data);
            }
        } catch (error) {
            console.error('Error escribiendo a la salida:', error);
        }
    }

    getCharFromColor(color) {
        switch (color) {
            case "#111111": return "  ";
            case "#262626": return "░░";
            case "#404040": return "▒▒";
            case "#767676": return "▓▓";
            case "#c7c7c7": 
            case "#929292":
            case "#bbbbbb":
            case "#9b9b9b":
            case "#a2a2a2": return "██";
            default: return "  ";
        }
    }

    getSystemLoad() {
        // Obtener métricas del sistema
        const cpuLoad = this.getCpuUsage();
        const memoryUsage = 1 - (os.freemem() / os.totalmem());
        const cpuDelta = Math.abs(cpuLoad - this.lastCpuUsage);
        
        this.lastCpuUsage = cpuLoad;
        
        return {
            cpuLoad: Math.min(cpuLoad, 1),
            memoryUsage,
            cpuDelta
        };
    }

    updateTraceSquares() {
        const currentTime = Date.now();
        const systemLoad = this.getSystemLoad();
        
        // Ajustar la probabilidad basada en la carga del sistema
        const baseChance = 0.15;
        const loadFactor = Math.max(0.1, Math.min(1, systemLoad.cpuLoad));
        const probability = baseChance * loadFactor;

        // Añadir cuadros basados en la actividad del sistema
        if (this.traceSquares.size < this.maxTraceSquares && Math.random() < probability) {
            // Seleccionar índice basado en uso de memoria
            const memorySection = Math.floor(systemLoad.memoryUsage * this.width);
            const startRange = memorySection * this.height;
            const endRange = (memorySection + 1) * this.height;
            const randomIndex = startRange + Math.floor(Math.random() * (endRange - startRange));

            this.traceSquares.add({
                index: randomIndex % (this.width * this.height),
                startTime: currentTime,
                intensity: systemLoad.cpuDelta
            });
        }

        // Actualizar duración del fade basado en la carga
        const dynamicDuration = this.TRACE_DURATION * (1 + systemLoad.cpuLoad);

        // Eliminar cuadros expirados
        for (const trace of this.traceSquares) {
            const age = (currentTime - trace.startTime) / dynamicDuration;
            if (age > 1) {
                this.traceSquares.delete(trace);
            }
        }
    }

    getFadeChar(age, intensity = 1) {
        // Ajustar la transición basada en la intensidad
        const fadeProgress = (Math.cos(age * Math.PI) * 0.5 + 0.5) * intensity;
        const fadeIndex = Math.floor(fadeProgress * (this.fadeSteps.length - 1));
        return this.fadeSteps[Math.min(fadeIndex, this.fadeSteps.length - 1)].repeat(2);
    }

    renderFrame() {
        this.updateTraceSquares();
        let output = '\n';

        for (let i = 0; i < this.height; i++) {
            for (let j = 0; j < this.width; j++) {
                const index = i * this.width + j;
                const currentIndex = this.currentPositions[index];
                const baseColor = GRID_COLORS[currentIndex];
                
                const isTrace = Array.from(this.traceSquares).find(trace => trace.index === index);
                
                if (isTrace) {
                    const age = (Date.now() - isTrace.startTime) / this.TRACE_DURATION;
                    output += this.getFadeChar(age, isTrace.intensity);
                } else {
                    output += this.getCharFromColor(baseColor);
                }
            }
            output += '\n';
        }

        return output;
    }

    renderToTTY() {
        try {
            const frame = this.renderFrame();
            const output = CONFIG.clearCommand + frame;
            this.writeToOutput(output);
        } catch (error) {
            console.error('Error rendering:', error);
        }
    }

    shuffleArray(array) {
        const newArray = [...array];
        for (let i = newArray.length - 1; i > 0; i--) {
            const j = Math.floor(Math.random() * (i + 1));
            [newArray[i], newArray[j]] = [newArray[j], newArray[i]];
        }
        return newArray;
    }

    animateGridScramble(isEncrypting) {
        if (this.isAnimating) return;
        this.isAnimating = true;

        try {
            const originalPositions = [...Array(this.width * this.height)].map((_, i) => i);
            const targetPositions = isEncrypting ? 
                this.shuffleArray(originalPositions) : 
                [...originalPositions];

            this.currentPositions = targetPositions;
        } catch (error) {
            console.error('Error en animación:', error);
        } finally {
            this.isAnimating = false;
        }
    }
    
    cleanup() {
        try {
            this.writeToOutput(CONFIG.resetCommand);
            
            if (this.ttyStream) {
                this.ttyStream.end();
                this.ttyStream = null;
            }
        } catch (error) {
            console.error('Error durante limpieza:', error);
        }
    }
}

// Crear el servidor
const server = http.createServer((req, res) => {
    const renderer = new MDSRenderer();
    res.writeHead(200, { 'Content-Type': 'text/plain; charset=utf-8' });
    res.end(renderer.renderFrame());
});

// Iniciar el servidor y mostrar el grid con animación
const PORT = 3000;
let renderer;
let renderInterval;

server.listen(PORT, () => {
    console.log(`Servidor corriendo en http://localhost:${PORT}`);
    renderer = new MDSRenderer();
    
    let lastRender = Date.now();
    
    // Verificar el acceso a TTY
    try {
        fs.accessSync(CONFIG.tty, fs.constants.W_OK);
        console.log('TTY accesible:', CONFIG.tty);
    } catch (err) {
        console.warn(`No se puede acceder a TTY (${CONFIG.tty}):`, err.message);
        console.log('Usando stdout como alternativa');
    }
    
    // Demostración de uso del método animateGridScramble
    setInterval(() => {
        // Alternar entre encriptado y normal cada 30 segundos
        renderer.animateGridScramble(Math.random() > 0.5);
    }, 30000);
    
    renderInterval = setInterval(() => {
        const now = Date.now();
        const systemLoad = renderer.getSystemLoad();
        const deltaTime = now - lastRender;
        
        // Ajustar el intervalo de actualización basado en la carga
        const targetFPS = Math.max(1, CONFIG.targetFps - (systemLoad.cpuLoad * 3));
        const targetFrameTime = 1000 / targetFPS;
        
        if (deltaTime >= targetFrameTime) {
            renderer.renderToTTY();
            lastRender = now;
        }
    }, CONFIG.updateInterval);
});

console.log("Iniciando servidor MDS");
const statusInterval = setInterval(() => {
    console.log("Servidor corriendo...");
}, 1000);

// Función de limpieza para asegurar que se ejecuta antes de salir
function cleanup() {
    if (renderer) {
        renderer.cleanup();
    }
    
    if (renderInterval) {
        clearInterval(renderInterval);
    }
    
    if (statusInterval) {
        clearInterval(statusInterval);
    }
    
    // Esperar un poco para asegurar que se complete la limpieza
    setTimeout(() => {
        process.exit(0);
    }, 100);
}

// Manejar la salida limpia del programa
process.on('SIGINT', cleanup);
process.on('SIGTERM', cleanup);
process.on('exit', () => {
    if (renderer) {
        renderer.cleanup();
    }
});
