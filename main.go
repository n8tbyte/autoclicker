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

// Const & Win32 API constants
const (
	fileName        = "autoclicker-store.json"
	defaultDelaySec = 1.0

	// Windows Hotkey & Message constants
	wmHotkey = 0x0312
	vkF2     = 0x71
	vkF3     = 0x72
)

var (
	user32           = windows.NewLazySystemDLL("user32.dll")
	procRegisterHK   = user32.NewProc("RegisterHotKey")
	procUnregisterHK = user32.NewProc("UnregisterHotKey")
	procGetMessage   = user32.NewProc("GetMessageW")
)

type ClickPoint struct {
	X      int     `json:"x"`
	Y      int     `json:"y"`
	Button string  `json:"button"`
	Delay  float64 `json:"delay"`
}

type winMSG struct {
	HWnd    windows.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

// Recorder manages state and concurrency safety for recording click sequences
type Recorder struct {
	mu          sync.RWMutex
	isRecording bool
	recorded    []ClickPoint
}

func NewRecorder() *Recorder {
	return &Recorder{
		recorded: make([]ClickPoint, 0),
	}
}

func (r *Recorder) IsRecording() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.isRecording
}

func (r *Recorder) Start() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.isRecording = true
	r.recorded = []ClickPoint{}

	// Load existing points if available
	if data, err := os.ReadFile(fileName); err == nil {
		_ = json.Unmarshal(data, &r.recorded)
	}
	return len(r.recorded)
}

func (r *Recorder) Stop() ([]ClickPoint, error) {
	r.mu.Lock()
	r.isRecording = false
	dataToSave := make([]ClickPoint, len(r.recorded))
	copy(dataToSave, r.recorded)
	r.mu.Unlock()

	fileData, err := json.MarshalIndent(dataToSave, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to encode JSON: %w", err)
	}

	if err := os.WriteFile(fileName, fileData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return dataToSave, nil
}

func (r *Recorder) AddPoint(x, y int, button string, delay float64) (int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isRecording {
		return 0, false
	}

	r.recorded = append(r.recorded, ClickPoint{
		X:      x,
		Y:      y,
		Button: button,
		Delay:  delay,
	})
	return len(r.recorded), true
}

func main() {
	a := app.New()
	w := a.NewWindow("Auto Clicker (F2 Record | F3 Save Point)")
	w.Resize(fyne.NewSize(380, 220))

	statusLabel := widget.NewLabel("Status: Ready (Press F2 to start recording)")
	statusLabel.Alignment = fyne.TextAlignCenter

	recorder := NewRecorder()

	btnPlay := widget.NewButton("Play Saved Clicks", func() {
		go playClicks(statusLabel)
	})

	w.SetContent(container.NewVBox(
		widget.NewLabelWithStyle("Auto Clicker Recorder", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		statusLabel,
		btnPlay,
	))

	go listenHotkeys(recorder, statusLabel)

	w.ShowAndRun()
}

func listenHotkeys(recorder *Recorder, label *widget.Label) {
	procRegisterHK.Call(0, 1, 0, vkF2)
	procRegisterHK.Call(0, 2, 0, vkF3)

	defer func() {
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
			case 1:
				handleToggleRecording(recorder, label)
			case 2:
				handleAddPoint(recorder, label)
			}
		}
	}
}

func handleToggleRecording(recorder *Recorder, label *widget.Label) {
	if !recorder.IsRecording() {
		loadedCount := recorder.Start()
		updateLabel(label, fmt.Sprintf("Recording... (Loaded %d existing points)", loadedCount))
	} else {
		savedPoints, err := recorder.Stop()
		if err != nil {
			updateLabel(label, fmt.Sprintf("Error: %v", err))
			return
		}
		updateLabel(label, fmt.Sprintf("Saved (%d total points) to %s", len(savedPoints), fileName))
	}
}

func handleAddPoint(recorder *Recorder, label *widget.Label) {
	x, y := robotgo.Location()
	count, added := recorder.AddPoint(x, y, "left", defaultDelaySec)
	if added {
		updateLabel(label, fmt.Sprintf("Point %d added at (%d, %d)", count, x, y))
	}
}

func playClicks(label *widget.Label) {
	data, err := os.ReadFile(fileName)
	if err != nil {
		updateLabel(label, fmt.Sprintf("Error: %s not found", fileName))
		return
	}

	var points []ClickPoint
	if err := json.Unmarshal(data, &points); err != nil {
		updateLabel(label, "Error: Invalid JSON format")
		return
	}

	if len(points) == 0 {
		updateLabel(label, "Warning: No points in file")
		return
	}

	updateLabel(label, "Playing clicks...")

	for _, pt := range points {
		robotgo.Move(pt.X, pt.Y)
		robotgo.Click(pt.Button, false)

		delaySec := pt.Delay
		if delaySec <= 0 {
			delaySec = defaultDelaySec
		}

		time.Sleep(time.Duration(delaySec * float64(time.Second)))
	}

	updateLabel(label, "Done!")
}

// Helper to safely execute UI updates on the Fyne main thread
func updateLabel(label *widget.Label, text string) {
	fyne.Do(func() {
		label.SetText(text)
	})
}
