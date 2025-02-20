package main

import (
	"fmt"
	"log"
	"net/http"
	"syscall"
	"time"
	"unsafe"

	"github.com/gin-gonic/gin"
	"github.com/lxn/win"
)

var (
	hwnd      win.HWND
	startTime time.Time
	isVisible bool
	timer     *time.Ticker
	done      chan bool
)

const (
	className  = "GeneratorTimerClass"
	windowName = "Generator Timer"
)

func main() {
	// Initialize channels for cleanup
	done = make(chan bool)

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

	// Message loop
	var msg win.MSG
	for {
		if ret := win.GetMessage(&msg, 0, 0, 0); ret == 0 || ret == -1 {
			break
		}
		win.TranslateMessage(&msg)
		win.DispatchMessage(&msg)
	}

	// Cleanup when the message loop ends
	if timer != nil {
		timer.Stop()
	}
	close(done)
}

func wndProc(hwnd win.HWND, msg uint32, wparam, lparam uintptr) uintptr {
	switch msg {
	case win.WM_PAINT:
		if isVisible {
			var ps win.PAINTSTRUCT
			hdc := win.BeginPaint(hwnd, &ps)
			if hdc != 0 {
				drawTimer(hdc)
				win.EndPaint(hwnd, &ps)
			}
		}
		return 0
	case win.WM_CLOSE:
		win.DestroyWindow(hwnd)
		return 0
	case win.WM_SYSCOMMAND:
		if wparam == win.SC_MINIMIZE {
			// Allow minimizing but keep the window visible
			return win.DefWindowProc(hwnd, msg, wparam, lparam)
		}
	case win.WM_DESTROY:
		win.PostQuitMessage(0)
		return 0
	}
	return win.DefWindowProc(hwnd, msg, wparam, lparam)
}

func drawTimer(hdc win.HDC) {
	if !isVisible || startTime.IsZero() {
		return
	}

	elapsed := time.Since(startTime)
	hours := int(elapsed.Hours())
	minutes := int(elapsed.Minutes()) % 60
	seconds := int(elapsed.Seconds()) % 60
	timeStr := fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)

	font := win.LOGFONT{
		LfHeight:         -48,
		LfWidth:          0,
		LfWeight:         win.FW_BOLD,
		LfCharSet:        win.ANSI_CHARSET,
		LfOutPrecision:   win.OUT_TT_PRECIS,
		LfClipPrecision:  win.CLIP_DEFAULT_PRECIS,
		LfQuality:        win.CLEARTYPE_QUALITY,
		LfPitchAndFamily: win.DEFAULT_PITCH | win.FF_DONTCARE,
	}
	copy(font.LfFaceName[:], syscall.StringToUTF16("Arial"))

	hFont := win.CreateFontIndirect(&font)
	if hFont == 0 {
		return
	}
	defer win.DeleteObject(win.HGDIOBJ(hFont))

	oldFont := win.SelectObject(hdc, win.HGDIOBJ(hFont))
	defer win.SelectObject(hdc, oldFont)

	win.SetTextColor(hdc, 0)
	win.SetBkMode(hdc, win.TRANSPARENT)

	// Get window client area dimensions
	var rect win.RECT
	win.GetClientRect(hwnd, &rect)

	// Calculate center position for text
	timeStrPtr := syscall.StringToUTF16Ptr(timeStr)
	var size win.SIZE
	win.GetTextExtentPoint32(hdc, timeStrPtr, int32(len(timeStr)), &size)

	x := (rect.Right - rect.Left - int32(size.CX)) / 2
	y := (rect.Bottom - rect.Top - int32(size.CY)) / 2

	win.TextOut(
		hdc,
		x,
		y,
		timeStrPtr,
		int32(len(timeStr)),
	)
}

func updateTimer() {
	if timer != nil {
		timer.Stop()
	}

	timer = time.NewTicker(time.Second)

	go func() {
		for {
			select {
			case <-timer.C:
				if isVisible {
					win.InvalidateRect(hwnd, nil, true)
					win.UpdateWindow(hwnd)
				}
			case <-done:
				return
			}
		}
	}()
}

func startServer() {
	router := gin.Default()

	router.POST("/create", func(c *gin.Context) {
		startTime = time.Now()
		isVisible = true
		updateTimer()
		win.ShowWindow(hwnd, win.SW_SHOW)
		win.SetWindowPos(hwnd, win.HWND_TOPMOST, 0, 0, 0, 0, win.SWP_NOMOVE|win.SWP_NOSIZE)
		win.UpdateWindow(hwnd)
		c.JSON(http.StatusOK, gin.H{"message": "Timer started"})
	})

	router.POST("/reset", func(c *gin.Context) {
		startTime = time.Now()
		win.InvalidateRect(hwnd, nil, true)
		win.UpdateWindow(hwnd)
		c.JSON(http.StatusOK, gin.H{"message": "Timer reset"})
	})

	router.POST("/hide", func(c *gin.Context) {
		isVisible = false
		win.ShowWindow(hwnd, win.SW_HIDE)
		c.JSON(http.StatusOK, gin.H{"message": "Timer hidden"})
	})

	if err := router.Run(":1997"); err != nil {
		log.Fatal(err)
	}
}
