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
	vkF1     = 0x70
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
	hotkeys := map[uintptr]uintptr{
		3: vkF1, // F1 -> Play/Stop
		1: vkF2, // F2 -> Record
		2: vkF3, // F3 -> Add Point
	}

	for id, vk := range hotkeys {
		r, _, _ := procRegisterHK.Call(0, id, 0, vk)
		if r == 0 {
			updateLabel(label, "Error: Failed to register hotkeys (F1/F2/F3 in use)")
		}
		defer procUnregisterHK.Call(0, id)
	}

	var msg winMSG
	for {
		ret, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}

		if msg.Message == wmHotkey {
			switch msg.WParam {
			case 3:
				togglePlayStop(label, recorder)
			case 1:
				handleToggleRecording(recorder, label, onUpdateUI)
			case 2:
				handleAddPoint(recorder, label, onUpdateUI)
			}
		}
	}
}

func handleToggleRecording(recorder *Recorder, label *widget.Label, onUpdateUI func()) {
	if !recorder.IsRecording() {
		count := recorder.Start()
		updateLabel(label, fmt.Sprintf("Recording (Loaded %d existing points)", count))
	} else {
		points, err := recorder.Stop()
		if err != nil {
			updateLabel(label, fmt.Sprintf("Error: %v", err))
			return
		}
		updateLabel(label, fmt.Sprintf("Saved (%d total points) to %s", len(points), fileName))
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
