package main

import (
    "flag"
    "fmt"
    "math"
    "math/rand"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gdamore/tcell/v2"
    "github.com/gdamore/tcell/v2/encoding"
)

const (
    width            = 16
    height           = 16
    updateInterval   = 100 * time.Millisecond   // Reduced update frequency to save CPU
    traceInterval    = 5000 * time.Millisecond  // Increased for smoother transitions
    traceDuration    = 6000 * time.Millisecond  // Longer duration for smoother fades
    maxTraceSquares  = 2                        // Reduced max concurrent traces
    scrambleInterval = 60 * time.Second         // Longer interval between scrambles
    idleCPUThreshold = 0.3                      // CPU usage threshold for throttling
    defaultTTY       = "/dev/tty"               // Default TTY device
)

var gridColors = []string{
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
    "#111111", "#111111", "#111111", "#111111", "#111111", "#111111", "#111111", "#171717", "#171717", "#171717", "#2b2b2b", "#2b2b2b", "#222222", "#222222", "#3a3a3a", "#111111",
}

type TraceSquare struct {
    index     int
    startTime time.Time
    intensity float64
}

type MDSRenderer struct {
    traceSquares     map[int]*TraceSquare
    currentPositions []int
    isAnimating      bool
    colorStyleCache  map[string]tcell.Style     // Cache for color styles
    charCache       map[string][]rune          // Cache for character representations
    lastUpdate      time.Time                  // Last update timestamp
    ttyDevice       string                     // TTY device for output
}

// Helper function to convert hex color to basic terminal color
func getBasicColorStyle(hexColor string) tcell.Style {
    style := tcell.StyleDefault
    
    // Map hex colors to basic terminal colors
    switch hexColor {
    case "#111111", "#0f0f0f", "#101010", "#131313":
        return style.Background(tcell.ColorBlack)
    case "#262626", "#1d1d1d", "#171717", "#1a1a1a", "#1b1b1b", "#1f1f1f", "#222222":
        return style.Background(tcell.ColorNavy)
    case "#404040", "#3a3a3a", "#3c3c3c", "#3e3e3e", "#3f3f3f":
        return style.Background(tcell.ColorBlue)
    case "#5b5b5b", "#5d5d5d", "#5e5e5e", "#515151":
        return style.Background(tcell.ColorDarkGray)
    case "#767676", "#727272":
        return style.Background(tcell.ColorGray)
    case "#929292", "#9b9b9b", "#a2a2a2", "#bbbbbb", "#c7c7c7":
        return style.Background(tcell.ColorSilver)
    default:
        return style.Background(tcell.ColorBlack)
    }
}

// Helper function to convert hex color to grayscale characters
func getCharsFromColor(hexColor string) []rune {
    switch hexColor {
    case "#111111", "#0f0f0f", "#101010", "#131313":
        return []rune{' ', ' '}
    case "#262626", "#1d1d1d", "#171717", "#1a1a1a", "#1b1b1b", "#1f1f1f", "#222222":
        return []rune{'░', '░'}
    case "#404040", "#3a3a3a", "#3c3c3c", "#3e3e3e", "#3f3f3f":
        return []rune{'▒', '▒'}
    case "#767676", "#5b5b5b", "#5d5d5d", "#5e5e5e", "#515151":
        return []rune{'▓', '▓'}
    case "#929292", "#9b9b9b", "#a2a2a2", "#bbbbbb", "#c7c7c7":
        return []rune{'█', '█'}
    default:
        return []rune{' ', ' '}
    }
}

// Helper function to get fade character based on progress
func getFadeChar(progress float64) []rune {
    fadeSteps := [][]rune{{'█', '█'}, {'▓', '▓'}, {'▒', '▒'}, {'░', '░'}, {' ', ' '}}
    fadeIndex := int(progress * float64(len(fadeSteps)-1))
    return fadeSteps[fadeIndex]
}

// Helper function for terminals with minimal color support - kept for compatibility
func getMonochromeStyle(hexColor string) tcell.Style {
    style := tcell.StyleDefault
    
    // Use brightness levels for monochrome display
    switch hexColor {
    case "#111111", "#0f0f0f", "#101010", "#131313", "#171717", "#1a1a1a", "#1b1b1b", "#1d1d1d", "#1f1f1f":
        return style.Background(tcell.ColorBlack)
    case "#222222", "#232323", "#242424", "#262626", "#272727", "#2b2b2b":
        return style.Background(tcell.ColorBlack)
    default:
        // For brighter colors, use gray
        return style.Background(tcell.ColorGray)
    }
}

func NewMDSRenderer(ttyDevice string) *MDSRenderer {
    r := &MDSRenderer{
        traceSquares:     make(map[int]*TraceSquare),
        currentPositions: make([]int, width*height),
        colorStyleCache:  make(map[string]tcell.Style),
        charCache:        make(map[string][]rune),
        lastUpdate:       time.Now(),
        ttyDevice:        ttyDevice,
    }
    for i := range r.currentPositions {
        r.currentPositions[i] = i
    }
    return r
}

func (r *MDSRenderer) updateTraceSquares() {
    now := time.Now()
    
    // Only process expired traces if we have any
    if len(r.traceSquares) > 0 {
        for index, trace := range r.traceSquares {
            age := now.Sub(trace.startTime)
            if age > traceDuration {
                delete(r.traceSquares, index)
            }
        }
    }

    // Only add new traces if we're below the limit and with reduced probability
    if len(r.traceSquares) < maxTraceSquares && rand.Float64() < 0.05 { // Further reduced probability
        index := rand.Intn(width * height)
        
        // Reuse existing trace if possible
        if trace, exists := r.traceSquares[index]; exists {
            trace.startTime = now
            trace.intensity = 0.6 + rand.Float64() * 0.2 // Even more subtle intensity
        } else {
            r.traceSquares[index] = &TraceSquare{
                index:     index,
                startTime: now,
                intensity: 0.6 + rand.Float64() * 0.2,
            }
        }
    }
}

// Restore the grid to its original positions
func (r *MDSRenderer) restoreOriginalPositions() {
    if originalPositions == nil {
        return
    }
    
    // Gradually restore original positions to make it visually smooth
    r.isAnimating = true
    defer func() { r.isAnimating = false }()
    
    // Copy original positions back to current positions
    for i := range r.currentPositions {
        r.currentPositions[i] = originalPositions[i]
    }
}





// Store original positions for restoration
var originalPositions []int

func (r *MDSRenderer) animateGridScramble() {
    if r.isAnimating {
        return
    }
    r.isAnimating = true
    defer func() { r.isAnimating = false }()
    
    // Initialize originalPositions if needed (only once)
    if originalPositions == nil {
        originalPositions = make([]int, width*height)
        for i := range originalPositions {
            originalPositions[i] = i
        }
    }
    
    // Reuse existing slice if possible to reduce allocations
    static := make([]int, len(r.currentPositions))
    copy(static, r.currentPositions)
    
    // Apply a minimal shuffle that only affects a small number of positions
    // This creates visual interest with minimal computation
    for i := 0; i < len(static)/8; i++ { // Only shuffle 12.5% of positions
        idx1 := rand.Intn(len(static))
        idx2 := rand.Intn(len(static))
        static[idx1], static[idx2] = static[idx2], static[idx1]
    }
    
    // Apply the scrambled positions directly
    copy(r.currentPositions, static)
    
    // Schedule restoration of original positions without creating a new goroutine
    // This avoids goroutine overhead for a simple timer
    time.AfterFunc(2*time.Second, func() {
        if !r.isAnimating { // Only restore if not in another animation
            r.restoreOriginalPositions()
        }
    })
}

func (r *MDSRenderer) drawGrid(screen tcell.Screen) {
    // Check if enough time has passed since last update
    if time.Since(r.lastUpdate) < updateInterval {
        time.Sleep(time.Until(r.lastUpdate.Add(updateInterval)))
    }
    r.lastUpdate = time.Now()

    screenWidth, screenHeight := screen.Size()
    
    // Calculate the target size to occupy 1/9 of the screen
    targetWidth := screenWidth / 3
    targetHeight := screenHeight / 3
    
    // Calculate scaling factors while maintaining aspect ratio
    scaleX := float64(targetWidth) / float64(width*2)
    scaleY := float64(targetHeight) / float64(height)
    scale := math.Min(scaleX, scaleY)
    
    // Calculate actual dimensions after scaling
    actualWidth := int(float64(width*2) * scale)
    actualHeight := int(float64(height) * scale)
    
    // Position in top-left corner with a small margin
    startX := 2 // Small margin from left
    startY := 1 // Small margin from top

    // Get terminal color capabilities
    colorMode := screen.Colors()

    // Calculate step sizes for responsive grid
    stepX := float64(width*2) / float64(actualWidth)
    stepY := float64(height) / float64(actualHeight)

    for y := 0; y < actualHeight; y++ {
        gridY := int(float64(y) * stepY)
        if gridY >= height {
            continue
        }
        
        for x := 0; x < actualWidth; x++ {
            gridX := int(float64(x) * stepX / 2)
            if gridX >= width {
                continue
            }
            
            index := gridY*width + gridX
            currentIndex := r.currentPositions[index]
            color := gridColors[currentIndex]
            
            // Use cached style if available
            style, exists := r.colorStyleCache[color]
            if !exists {
                if colorMode >= 256 {
                    style = tcell.StyleDefault.Background(tcell.GetColor(color))
                } else if colorMode >= 8 {
                    style = getBasicColorStyle(color)
                } else {
                    style = getMonochromeStyle(color)
                }
                r.colorStyleCache[color] = style
            }

            if trace, exists := r.traceSquares[index]; exists {
                age := time.Since(trace.startTime)
                fadeProgress := math.Cos((age.Seconds() / traceDuration.Seconds()) * math.Pi) * 0.5 + 0.5
                fadeProgress = fadeProgress * trace.intensity
                fadeChar := getFadeChar(fadeProgress)
                
                screen.SetContent(startX+x, startY+y, fadeChar[0], nil, style)
            } else {
                chars, exists := r.charCache[color]
                if !exists {
                    chars = getCharsFromColor(color)
                    r.charCache[color] = chars
                }
                
                screen.SetContent(startX+x, startY+y, chars[0], nil, style)
            }
        }
    }
}

func main() {
    // Parse command line flags
    ttyDevice := flag.String("tty", defaultTTY, "TTY device to use for output (e.g., /dev/tty1)")
    flag.Parse()

    // Register UTF-8 encoding to ensure proper character display
    encoding.Register()
    
    // Initialize screen
    screen, err := tcell.NewScreen()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error creating screen: %v\n", err)
        os.Exit(1)
    }
    
    if err := screen.Init(); err != nil {
        fmt.Fprintf(os.Stderr, "Error initializing screen: %v\n", err)
        os.Exit(1)
    }
    
    // Set terminal title and clear screen
    fmt.Print("\033]0;MDS Grid Renderer\007")
    screen.Clear()

    renderer := NewMDSRenderer(*ttyDevice)

    quit := make(chan struct{})
    go func() {
        for {
            ev := screen.PollEvent()
            switch ev := ev.(type) {
            case *tcell.EventKey:
                if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
                    close(quit)
                    return
                }
            case *tcell.EventResize:
                screen.Sync()
            }
        }
    }()

    updateTicker := time.NewTicker(updateInterval)
    scrambleTicker := time.NewTicker(scrambleInterval)
    
    // Add adaptive timing for updates
    var skipCounter int
    var lastActivity time.Time = time.Now()

    go func() {
        for {
            select {
            case <-quit:
                return
            case <-updateTicker.C:
                // Check if there's any activity to render
                hasActivity := len(renderer.traceSquares) > 0 || renderer.isAnimating
                
                // If no activity, throttle updates to save CPU
                if !hasActivity {
                    skipCounter++
                    // Skip some frames during idle periods
                    if skipCounter < 5 { // Only render every 5th frame when idle
                        continue
                    }
                    // If idle for more than 10 seconds, sleep longer
                    if time.Since(lastActivity) > 10*time.Second {
                        time.Sleep(100 * time.Millisecond) // Additional sleep during long idle periods
                    }
                } else {
                    lastActivity = time.Now()
                    skipCounter = 0
                }
                
                renderer.updateTraceSquares()
                screen.Clear()
                renderer.drawGrid(screen)
                screen.Show()
            case <-scrambleTicker.C:
                renderer.animateGridScramble()
                lastActivity = time.Now() // Reset idle timer on scramble
            }
        }
    }()

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    select {
    case <-quit:
    case <-sigChan:
        close(quit)
    }
    <-quit

    screen.Fini()
}
