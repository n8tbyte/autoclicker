package main

import (
	"fmt"
	"unsafe"

	"fyne.io/fyne/v2/widget"
	"github.com/go-vgo/robotgo"
	"golang.org/x/sys/windows"
)

const (
	wmHotkey = 0x0312
	vkF1     = 0x70 // F1 Key Code
	vkF2     = 0x71
	vkF3     = 0x72
)

var (
	user32           = windows.NewLazySystemDLL("user32.dll")
	procRegisterHK   = user32.NewProc("RegisterHotKey")
	procUnregisterHK = user32.NewProc("UnregisterHotKey")
	procGetMessage   = user32.NewProc("GetMessageW")
)

type winMSG struct {
	HWnd    windows.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

func listenHotkeys(recorder *Recorder, label *widget.Label, onUpdateUI func()) {
	// Register F1 (ID: 3), F2 (ID: 1), F3 (ID: 2)
	r0, _, _ := procRegisterHK.Call(0, 3, 0, vkF1)
	r1, _, _ := procRegisterHK.Call(0, 1, 0, vkF2)
	r2, _, _ := procRegisterHK.Call(0, 2, 0, vkF3)

	if r0 == 0 || r1 == 0 || r2 == 0 {
		updateLabel(label, "Error: Failed to register hotkeys (F1/F2/F3 already in use)")
	}

	defer func() {
		procUnregisterHK.Call(0, 3)
		procUnregisterHK.Call(0, 1)
		procUnregisterHK.Call(0, 2)
	}()

	var msg winMSG
	for {
		ret, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}

		if msg.Message == wmHotkey {
			switch msg.WParam {
			case 3: // F1
				togglePlayStop(label, recorder)
			case 1: // F2
				handleToggleRecording(recorder, label, onUpdateUI)
			case 2: // F3
				handleAddPoint(recorder, label, onUpdateUI)
			}
		}
	}
}

func handleToggleRecording(recorder *Recorder, label *widget.Label, onUpdateUI func()) {
	if !recorder.IsRecording() {
		loadedCount := recorder.Start()
		updateLabel(label, fmt.Sprintf("Recording (Loaded %d existing points)", loadedCount))
	} else {
		savedPoints, err := recorder.Stop()
		if err != nil {
			updateLabel(label, fmt.Sprintf("Error: %v", err))
			return
		}
		updateLabel(label, fmt.Sprintf("Saved (%d total points) to %s", len(savedPoints), fileName))
	}
	onUpdateUI()
}

func handleAddPoint(recorder *Recorder, label *widget.Label, onUpdateUI func()) {
	x, y := robotgo.Location()
	count, added := recorder.AddPoint(x, y, "left", defaultDelaySec)
	if added {
		updateLabel(label, fmt.Sprintf("Point %d added at (%d, %d)", count, x, y))
		onUpdateUI()
	}
}
