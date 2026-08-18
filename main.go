package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/go-vgo/robotgo"
	"golang.org/x/sys/windows"
)

type ClickPoint struct {
	X      int     `json:"x"`
	Y      int     `json:"y"`
	Button string  `json:"button"`
	Delay  float64 `json:"delay"`
}

const filename = "autoclicker-store.json"

var (
	user32           = windows.NewLazySystemDLL("user32.dll")
	procRegisterHK   = user32.NewProc("RegisterHotKey")
	procUnregisterHK = user32.NewProc("UnregisterHotKey")
	procGetMessage   = user32.NewProc("GetMessageW")

	isRecording bool
	recorded    []ClickPoint
	mu          sync.Mutex
)

type MSG struct {
	HWnd    windows.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

func main() {
	a := app.New()
	w := a.NewWindow("Auto Clicker (F2 Start/Stop Record | F3 Save Point)")
	w.Resize(fyne.NewSize(380, 220))

	statusLabel := widget.NewLabel("Status: Ready (Press F2 to start recording)")
	statusLabel.Alignment = fyne.TextAlignCenter

	btnPlay := widget.NewButton("Play saved clicks", func() {
		go playClicks(statusLabel)
	})

	w.SetContent(container.NewVBox(
		widget.NewLabelWithStyle("Auto Clicker Recorder", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		statusLabel,
		btnPlay,
	))

	go listenHotkeys(statusLabel)

	w.ShowAndRun()
}

func listenHotkeys(label *widget.Label) {
	procRegisterHK.Call(0, 1, 0, 0x71)
	procRegisterHK.Call(0, 2, 0, 0x72)

	defer func() {
		procUnregisterHK.Call(0, 1)
		procUnregisterHK.Call(0, 2)
	}()

	var msg MSG
	for {
		ret, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if ret == 0 {
			break
		}

		if msg.Message == 0x0312 {
			switch msg.WParam {
			case 1:
				toggleRecording(label)
			case 2:
				addClickPoint(label)
			}
		}
	}
}

func toggleRecording(label *widget.Label) {
	mu.Lock()
	if !isRecording {
		isRecording = true
		recorded = []ClickPoint{}
		mu.Unlock()

		fyne.Do(func() {
			label.SetText("Recording started... (Press F3 to mark point, F2 to stop)")
		})
	} else {
		isRecording = false
		dataToSave := recorded
		mu.Unlock()

		fileData, err := json.MarshalIndent(dataToSave, "", "  ")

		fyne.Do(func() {
			if err == nil {
				os.WriteFile(filename, fileData, 0644)
				label.SetText(fmt.Sprintf("Saved (%d points) to %s", len(dataToSave), filename))
			} else {
				label.SetText("Error saving file")
			}
		})
	}
}

func addClickPoint(label *widget.Label) {
	mu.Lock()
	defer mu.Unlock()

	if !isRecording {
		return
	}

	x, y := robotgo.Location()
	recorded = append(recorded, ClickPoint{X: x, Y: y, Button: "left", Delay: 1.0})
	count := len(recorded)

	fyne.Do(func() {
		label.SetText(fmt.Sprintf("Point %d added at (%d, %d)", count, x, y))
	})
}

func playClicks(label *widget.Label) {
	data, err := os.ReadFile(filename)
	if err != nil {
		fyne.Do(func() { label.SetText(fmt.Sprintf("Error: %s not found", filename)) })
		return
	}

	var points []ClickPoint
	if err := json.Unmarshal(data, &points); err != nil {
		fyne.Do(func() { label.SetText("Error: Invalid JSON format") })
		return
	}

	if len(points) == 0 {
		fyne.Do(func() { label.SetText("Warning: No points in file") })
		return
	}

	fyne.Do(func() {
		label.SetText("Playing clicks...")
	})

	for _, pt := range points {
		robotgo.Move(pt.X, pt.Y)
		robotgo.Click(pt.Button, false)

		delaySec := pt.Delay
		if delaySec <= 0 {
			delaySec = 1.0
		}

		sleepDuration := time.Duration(delaySec * float64(time.Second))
		time.Sleep(sleepDuration)
	}

	fyne.Do(func() { label.SetText("Done!") })
}
