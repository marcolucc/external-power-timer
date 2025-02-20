package main

import (
    "fmt"
    "github.com/gin-gonic/gin"
    "github.com/lxn/win"
    "log"
    "net/http"
    "syscall"
    "time"
    "unsafe"
)

var (
    hwnd      win.HWND
    startTime time.Time
    isVisible bool
)

const (
    className  = "GeneratorTimerClass"
    windowName = "Generator Timer"
)

func main() {
    // Start HTTP server in a goroutine
    go startServer()

    // Initialize and register the window class
    hinst := win.GetModuleHandle(nil)
    if hinst == 0 {
        log.Fatal("GetModuleHandle failed")
    }

    wcx := &win.WNDCLASSEX{
        CbSize:        uint32(unsafe.Sizeof(win.WNDCLASSEX{})),
        LpfnWndProc:   syscall.NewCallback(wndProc),
        HInstance:     hinst,
        HbrBackground: win.HBRUSH(win.GetStockObject(win.WHITE_BRUSH)),
        LpszClassName: syscall.StringToUTF16Ptr(className),
    }

    if atom := win.RegisterClassEx(wcx); atom == 0 {
        log.Fatal("RegisterClassEx failed")
    }

    // Create the window (initially hidden)
    hwnd = win.CreateWindowEx(
        0,
        syscall.StringToUTF16Ptr(className),
        syscall.StringToUTF16Ptr(windowName),
        win.WS_OVERLAPPEDWINDOW,
        win.CW_USEDEFAULT,
        win.CW_USEDEFAULT,
        300,
        150,
        0,
        0,
        hinst,
        nil,
    )

    if hwnd == 0 {
        log.Fatal("CreateWindowEx failed")
    }

    // Start message loop
    var msg win.MSG
    for {
        if ret := win.GetMessage(&msg, 0, 0, 0); ret == 0 || ret == -1 {
            break
        }
        win.TranslateMessage(&msg)
        win.DispatchMessage(&msg)
    }
}

func wndProc(hwnd win.HWND, msg uint32, wparam, lparam uintptr) uintptr {
    switch msg {
    case win.WM_PAINT:
        if isVisible {
            var ps win.PAINTSTRUCT
            hdc := win.BeginPaint(hwnd, &ps)
            drawTimer(hdc)
            win.EndPaint(hwnd, &ps)
        }
        return 0
    case win.WM_CLOSE:
        // Prevent the user from closing the window
        return 0
    case win.WM_SYSCOMMAND:
        if wparam == win.SC_MINIMIZE {
            // Prevent minimizing the window
            return 0
        }
    case win.WM_DESTROY:
        win.PostQuitMessage(0)
        return 0
    default:
        return win.DefWindowProc(hwnd, msg, wparam, lparam)
    }
	return 0
}




func drawTimer(hdc win.HDC) {
    elapsed := time.Since(startTime)
    hours := int(elapsed.Hours())
    minutes := int(elapsed.Minutes()) % 60
    seconds := int(elapsed.Seconds()) % 60
    timeStr := fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)

    // Create a logical font
    font := win.LOGFONT{
        LfHeight:         -48, // Negative value for character height
        LfWidth:          0,   // Match height ratio
        LfWeight:         win.FW_BOLD,
        LfCharSet:        win.ANSI_CHARSET,
        LfOutPrecision:   win.OUT_TT_PRECIS,
        LfClipPrecision:  win.CLIP_DEFAULT_PRECIS,
        LfQuality:        win.CLEARTYPE_QUALITY,
        LfPitchAndFamily: win.DEFAULT_PITCH | win.FF_DONTCARE,
    }
    
    // Copy "Arial" into the lfFaceName array
    copy(font.LfFaceName[:], syscall.StringToUTF16("Arial"))

    // Create the font using CreateFontIndirect
    hFont := win.CreateFontIndirect(&font)
    if hFont == 0 {
        return
    }
    defer win.DeleteObject(win.HGDIOBJ(hFont))

    // Select the new font into the DC
    win.SelectObject(hdc, win.HGDIOBJ(hFont))
    
    // Set text color and background mode
    win.SetTextColor(hdc, 0)
    win.SetBkMode(hdc, win.TRANSPARENT)

    // Draw the text
    win.TextOut(
        hdc,
        50,  // x position
        30,  // y position
        syscall.StringToUTF16Ptr(timeStr),
        int32(len(timeStr)),
    )
}

func startServer() {
    router := gin.Default()

    // API endpoint to create and start timer
    router.POST("/create", func(c *gin.Context) {
        startTime = time.Now()
        isVisible = true
        win.ShowWindow(hwnd, win.SW_SHOW)
        win.SetWindowPos(hwnd, win.HWND_TOPMOST, 0, 0, 0, 0, win.SWP_NOMOVE|win.SWP_NOSIZE)
        win.UpdateWindow(hwnd)

        // Start updating the timer display
        go updateTimer()

        c.JSON(http.StatusOK, gin.H{"message": "Timer started"})
    })

    // API endpoint to reset timer
    router.POST("/reset", func(c *gin.Context) {
        startTime = time.Now()
        win.InvalidateRect(hwnd, nil, true)
        c.JSON(http.StatusOK, gin.H{"message": "Timer reset"})
    })

    // API endpoint to hide timer
    router.POST("/hide", func(c *gin.Context) {
        isVisible = false
        win.ShowWindow(hwnd, win.SW_HIDE)
        c.JSON(http.StatusOK, gin.H{"message": "Timer hidden"})
    })

    if err := router.Run(":1997"); err != nil {
        log.Fatal(err)
    }
}


func updateTimer() {
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    for range ticker.C {
        if !isVisible {
            continue
        }
        win.InvalidateRect(hwnd, nil, true)
    }
}