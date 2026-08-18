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
	"fyne.io/fyne/v2/dialog"
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

func (r *Recorder) GetRecorded() []ClickPoint {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pts := make([]ClickPoint, len(r.recorded))
	copy(pts, r.recorded)
	return pts
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

	if err := savePointsToJSON(dataToSave); err != nil {
		return nil, err
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
	w := a.NewWindow("Auto Clicker")
	w.Resize(fyne.NewSize(200, 400))

	statusLabel := widget.NewLabel("Status: Ready (Press F2 to start recording)")
	statusLabel.Alignment = fyne.TextAlignCenter

	recorder := NewRecorder()

	var displayPoints []ClickPoint
	selectedIndex := -1 // ตัวแปรเก็บตำแหน่งรายการที่ถูกเลือกใน List

	pointsList := widget.NewList(
		func() int {
			return len(displayPoints)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template Point Item")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			pt := displayPoints[i]
			o.(*widget.Label).SetText(fmt.Sprintf("%d. X: %d, Y: %d | Button: %s | Delay: %.1fs", i+1, pt.X, pt.Y, pt.Button, pt.Delay))
		},
	)

	// เมื่อคลิกเลือกรายการใน List
	pointsList.OnSelected = func(id widget.ListItemID) {
		selectedIndex = int(id)
	}
	pointsList.OnUnselected = func(id widget.ListItemID) {
		selectedIndex = -1
	}

	// ฟังก์ชันรีเฟรชข้อมูลใน List
	refreshUIList := func() {
		fyne.Do(func() {
			if recorder.IsRecording() {
				displayPoints = recorder.GetRecorded()
			} else {
				displayPoints = loadPointsFromJSON()
			}
			selectedIndex = -1
			pointsList.UnselectAll()
			pointsList.Refresh()
		})
	}

	displayPoints = loadPointsFromJSON()

	btnPlay := widget.NewButton("Play Saved Clicks", func() {
		go playClicks(statusLabel)
	})

	btnReload := widget.NewButton("Reload JSON", func() {
		refreshUIList()
		updateLabel(statusLabel, fmt.Sprintf("Reloaded %d points from JSON", len(displayPoints)))
	})

	// ปุ่มลบรายการที่เลือก
	btnDeleteSelected := widget.NewButton("Delete Selected", func() {
		if recorder.IsRecording() {
			updateLabel(statusLabel, "Cannot delete while recording!")
			return
		}
		if selectedIndex < 0 || selectedIndex >= len(displayPoints) {
			updateLabel(statusLabel, "Please select an item to delete")
			return
		}

		// ลบรายการที่ selectedIndex ออกจาก Array
		displayPoints = append(displayPoints[:selectedIndex], displayPoints[selectedIndex+1:]...)
		if err := savePointsToJSON(displayPoints); err != nil {
			updateLabel(statusLabel, fmt.Sprintf("Error saving file: %v", err))
			return
		}

		updateLabel(statusLabel, "Deleted selected item")
		refreshUIList()
	})

	// ปุ่มลบทั้งหมด (พร้อม Popup ยืนยัน)
	btnClearAll := widget.NewButton("Clear All", func() {
		if recorder.IsRecording() {
			updateLabel(statusLabel, "Cannot clear while recording!")
			return
		}
		if len(displayPoints) == 0 {
			updateLabel(statusLabel, "List is already empty")
			return
		}

		dialog.ShowConfirm("Confirm Clear All", "Are you sure you want to delete all points?", func(confirmed bool) {
			if confirmed {
				displayPoints = []ClickPoint{}
				if err := savePointsToJSON(displayPoints); err != nil {
					updateLabel(statusLabel, fmt.Sprintf("Error saving file: %v", err))
					return
				}
				updateLabel(statusLabel, "All points cleared")
				refreshUIList()
			}
		}, w)
	})

	topBox := container.NewVBox(
		widget.NewLabelWithStyle("Auto Clicker Recorder", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		statusLabel,
		container.NewGridWithColumns(2, btnPlay, btnReload),
		widget.NewLabelWithStyle("Saved Click Points:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)

	bottomBox := container.NewGridWithColumns(2, btnDeleteSelected, btnClearAll)

	w.SetContent(container.NewBorder(
		topBox,
		bottomBox,
		nil,
		nil,
		pointsList,
	))

	go listenHotkeys(recorder, statusLabel, refreshUIList)

	w.ShowAndRun()
}

// ฟังก์ชันบันทึกข้อมูลลงไฟล์ JSON
func savePointsToJSON(points []ClickPoint) error {
	fileData, err := json.MarshalIndent(points, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	if err := os.WriteFile(fileName, fileData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

// ฟังก์ชันดึงข้อมูลจุดคลิกจากไฟล์ JSON
func loadPointsFromJSON() []ClickPoint {
	data, err := os.ReadFile(fileName)
	if err != nil {
		return []ClickPoint{}
	}

	var points []ClickPoint
	if err := json.Unmarshal(data, &points); err != nil {
		return []ClickPoint{}
	}

	return points
}

func listenHotkeys(recorder *Recorder, label *widget.Label, onUpdateUI func()) {
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
				handleToggleRecording(recorder, label, onUpdateUI)
			case 2:
				handleAddPoint(recorder, label, onUpdateUI)
			}
		}
	}
}

func handleToggleRecording(recorder *Recorder, label *widget.Label, onUpdateUI func()) {
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

func playClicks(label *widget.Label) {
	points := loadPointsFromJSON()

	if len(points) == 0 {
		updateLabel(label, "Warning: No points in file or file not found")
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
