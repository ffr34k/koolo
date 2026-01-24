package game

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/lxn/win"
)

const (
	RightButton MouseButton = win.MK_RBUTTON
	LeftButton  MouseButton = win.MK_LBUTTON

	ShiftKey ModifierKey = win.VK_SHIFT
	CtrlKey  ModifierKey = win.VK_CONTROL
)

type MouseButton uint
type ModifierKey byte

const pointerReleaseDelay = 150 * time.Millisecond

// Resolution validation constants
const (
	ExpectedWidth       = 1280
	ExpectedHeight      = 720
	ResizeTolerance     = 10
	ResizeWaitTime      = 500 * time.Millisecond
	MaxResizeRetries    = 3
	ResolutionCacheTime = 1 * time.Second // Skip re-check if validated within this time
)

// Resolution cache to avoid per-click overhead
var (
	lastResolutionCheck time.Time
	resolutionValid     bool
	resolutionMu        sync.Mutex
)

// MovePointer moves the mouse to the requested position, x and y should be the final position based on
// pixels shown in the screen. Top-left corner is 0,0
func (hid *HID) MovePointer(x, y int) {
	hid.gr.updateWindowPositionData()
	x = hid.gr.WindowLeftX + x
	y = hid.gr.WindowTopY + y

	hid.gi.CursorPos(x, y)
	lParam := calculateLparam(x, y)
	win.SendMessage(hid.gr.HWND, win.WM_NCHITTEST, 0, lParam)
	win.SendMessage(hid.gr.HWND, win.WM_SETCURSOR, 0x000105A8, 0x2010001)
	win.PostMessage(hid.gr.HWND, win.WM_MOUSEMOVE, 0, lParam)
}

// Click performs a single mouse click at the specified position.
// Returns an error if resolution validation fails after retries.
func (hid *HID) Click(btn MouseButton, x, y int) error {
	// Ensure correct resolution BEFORE moving pointer
	if err := hid.ensureCorrectResolution(); err != nil {
		return err
	}

	hid.MovePointer(x, y)
	x = hid.gr.WindowLeftX + x
	y = hid.gr.WindowTopY + y

	lParam := calculateLparam(x, y)
	buttonDown := uint32(win.WM_LBUTTONDOWN)
	buttonUp := uint32(win.WM_LBUTTONUP)
	if btn == RightButton {
		buttonDown = win.WM_RBUTTONDOWN
		buttonUp = win.WM_RBUTTONUP
	}

	win.SendMessage(hid.gr.HWND, buttonDown, 1, lParam)
	sleepTime := rand.Intn(keyPressMaxTime-keyPressMinTime) + keyPressMinTime
	time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	win.SendMessage(hid.gr.HWND, buttonUp, 1, lParam)

	return nil
}

// ensureCorrectResolution checks if the game window is at the expected resolution.
// If not, it attempts to resize the window up to MaxResizeRetries times.
// Uses caching to avoid per-click overhead.
// Returns an error if the resolution cannot be corrected.
func (hid *HID) ensureCorrectResolution() error {
	resolutionMu.Lock()
	defer resolutionMu.Unlock()

	// Use cached result if recently validated
	if resolutionValid && time.Since(lastResolutionCheck) < ResolutionCacheTime {
		return nil
	}

	for attempt := 0; attempt <= MaxResizeRetries; attempt++ {
		hid.gr.updateWindowPositionData()

		widthOk := abs(hid.gr.GameAreaSizeX-ExpectedWidth) <= ResizeTolerance
		heightOk := abs(hid.gr.GameAreaSizeY-ExpectedHeight) <= ResizeTolerance

		if widthOk && heightOk {
			resolutionValid = true
			lastResolutionCheck = time.Now()
			return nil
		}

		if attempt < MaxResizeRetries {
			hid.logger.Warn("Resolution mismatch, attempting resize",
				"attempt", attempt+1,
				"got", fmt.Sprintf("%dx%d", hid.gr.GameAreaSizeX, hid.gr.GameAreaSizeY),
				"expected", fmt.Sprintf("%dx%d", ExpectedWidth, ExpectedHeight))

			hid.gr.ForceResize(ExpectedWidth, ExpectedHeight)
			time.Sleep(ResizeWaitTime)
		}
	}

	resolutionValid = false
	err := fmt.Errorf("failed to set resolution after %d attempts: got %dx%d, expected %dx%d",
		MaxResizeRetries,
		hid.gr.GameAreaSizeX, hid.gr.GameAreaSizeY,
		ExpectedWidth, ExpectedHeight)

	hid.logger.Error("Resolution check failed - cannot proceed with UI clicks", "error", err)
	return err
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ClickWithModifier performs a click while holding a modifier key.
// Returns an error if resolution validation fails.
func (hid *HID) ClickWithModifier(btn MouseButton, x, y int, modifier ModifierKey) error {
	hid.gi.OverrideGetKeyState(byte(modifier))
	err := hid.Click(btn, x, y)
	hid.gi.RestoreGetKeyState()
	return err
}

func calculateLparam(x, y int) uintptr {
	return uintptr(y<<16 | x)
}
