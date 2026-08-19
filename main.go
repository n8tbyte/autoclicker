package main

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/go-vgo/robotgo"
)

var (
	playMu    sync.Mutex
	isPlaying bool
	stopChan  chan struct{}
)

func main() {
	a := app.New()
	w := a.NewWindow("Auto Clicker " + version)
	w.Resize(fyne.NewSize(300, 400))

	statusLabel := widget.NewLabel("F1: Play/Stop | F2: Record | F3: Add Point")
	recorder := NewRecorder()

	var pointsList *widget.List
	var refreshUIList func()

	pointsList = widget.NewList(
		func() int {
			return recorder.Len()
		},
		func() fyne.CanvasObject {
			labelNum := widget.NewLabel("")
			labelNum.Alignment = fyne.TextAlignCenter

			entryX := widget.NewEntry()
			entryY := widget.NewEntry()
			entryDelay := widget.NewEntry()

			btnDelete := widget.NewButton("X", nil)

			return container.NewGridWithColumns(5, labelNum, entryX, entryY, entryDelay, btnDelete)
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			pt, ok := recorder.GetAt(id)
			if !ok {
				return
			}

			grid := o.(*fyne.Container)

			labelNum := grid.Objects[0].(*widget.Label)
			entryX := grid.Objects[1].(*widget.Entry)
			entryY := grid.Objects[2].(*widget.Entry)
			entryDelay := grid.Objects[3].(*widget.Entry)
			btnDelete := grid.Objects[4].(*widget.Button)

			entryX.OnChanged = nil
			entryY.OnChanged = nil
			entryDelay.OnChanged = nil
			btnDelete.OnTapped = nil

			labelNum.SetText(fmt.Sprintf("%d", id+1))
			entryX.SetText(fmt.Sprintf("%d", pt.X))
			entryY.SetText(fmt.Sprintf("%d", pt.Y))
			entryDelay.SetText(fmt.Sprintf("%.1f", pt.Delay))

			entryX.OnChanged = func(val string) {
				if x, err := strconv.Atoi(val); err == nil {
					_ = recorder.UpdatePoint(id, func(p *ClickPoint) { p.X = x })
				}
			}

			entryY.OnChanged = func(val string) {
				if y, err := strconv.Atoi(val); err == nil {
					_ = recorder.UpdatePoint(id, func(p *ClickPoint) { p.Y = y })
				}
			}

			entryDelay.OnChanged = func(val string) {
				if d, err := strconv.ParseFloat(val, 64); err == nil {
					_ = recorder.UpdatePoint(id, func(p *ClickPoint) { p.Delay = d })
				}
			}

			btnDelete.OnTapped = func() {
				if err := recorder.DeletePoint(id); err != nil {
					updateLabel(statusLabel, err.Error())
					return
				}
				updateLabel(statusLabel, "Deleted point")
				refreshUIList()
			}
		},
	)

	refreshUIList = func() {
		fyne.Do(func() {
			pointsList.Refresh()
		})
	}

	btnPlay := widget.NewButton("Play", func() {
		togglePlayStop(statusLabel, recorder)
	})
	btnReload := widget.NewButton("Reload", func() {
		recorder.Load()
		refreshUIList()
		updateLabel(statusLabel, fmt.Sprintf("Reloaded %d points from CSV", recorder.Len()))
	})

	topBox := container.NewVBox(
		statusLabel,
		container.NewGridWithColumns(2, btnPlay, btnReload),
	)

	w.SetContent(container.NewBorder(
		topBox,
		nil,
		nil,
		nil,
		pointsList,
	))

	go listenHotkeys(recorder, statusLabel, refreshUIList)
	w.ShowAndRun()
}

func togglePlayStop(label *widget.Label, recorder *Recorder) {
	playMu.Lock()
	if isPlaying {
		// หากกำลังเล่นอยู่ ให้ส่งสัญญาณหยุด
		if stopChan != nil {
			close(stopChan)
			stopChan = nil
		}
		playMu.Unlock()
		updateLabel(label, "Stopping")
		return
	}

	isPlaying = true
	stopChan = make(chan struct{})
	ch := stopChan
	playMu.Unlock()

	go playClicks(label, recorder, ch)
}

func playClicks(label *widget.Label, recorder *Recorder, ch chan struct{}) {
	defer func() {
		playMu.Lock()
		isPlaying = false
		stopChan = nil
		playMu.Unlock()
	}()

	points := recorder.GetRecorded()
	if len(points) == 0 {
		updateLabel(label, "Warning: No points in memory or file empty")
		return
	}

	updateLabel(label, "Playing (Press F1 to stop)")
	for _, pt := range points {
		// ตรวจสอบสัญญาณหยุด
		select {
		case <-ch:
			updateLabel(label, "Stopped by user")
			return
		default:
		}

		robotgo.Move(pt.X, pt.Y)
		robotgo.Click(pt.Button, false)

		delaySec := pt.Delay
		if delaySec <= 0 {
			delaySec = defaultDelaySec
		}

		// รอตาม Delay โดยพร้อมรับสัญญาณหยุดทันที
		select {
		case <-ch:
			updateLabel(label, "Stopped by user")
			return
		case <-time.After(time.Duration(delaySec * float64(time.Second))):
		}
	}
	updateLabel(label, "Done")
}

func updateLabel(label *widget.Label, text string) {
	fyne.Do(func() {
		label.SetText(text)
	})
}
