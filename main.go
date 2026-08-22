package main

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
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

	statusLabel := widget.NewLabel("F1: Play/Stop | F2: Record/Stop | F3: Add Point")
	recorder := NewRecorder()

	var pointsList *widget.List
	refreshUIList := func() {
		fyne.Do(func() { pointsList.Refresh() })
	}

	pointsList = widget.NewList(
		recorder.Len,
		func() fyne.CanvasObject {
			lblNum := widget.NewLabel("")
			lblNum.Alignment = fyne.TextAlignCenter
			return container.NewGridWithColumns(5,
				lblNum,
				widget.NewEntry(),
				widget.NewEntry(),
				widget.NewEntry(),
				widget.NewButton("X", nil),
			)
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			pt, ok := recorder.GetAt(id)
			if !ok {
				return
			}

			grid := o.(*fyne.Container)
			lblNum := grid.Objects[0].(*widget.Label)
			entryX := grid.Objects[1].(*widget.Entry)
			entryY := grid.Objects[2].(*widget.Entry)
			entryDelay := grid.Objects[3].(*widget.Entry)
			btnDelete := grid.Objects[4].(*widget.Button)

			entryX.OnChanged, entryY.OnChanged, entryDelay.OnChanged, btnDelete.OnTapped = nil, nil, nil, nil

			lblNum.SetText(fmt.Sprintf("%d", id+1))
			entryX.SetText(strconv.Itoa(pt.X))
			entryY.SetText(strconv.Itoa(pt.Y))
			entryDelay.SetText(fmt.Sprintf("%.1f", pt.Delay))

			entryX.OnChanged = func(v string) {
				if x, err := strconv.Atoi(v); err == nil {
					_ = recorder.UpdatePoint(id, func(p *ClickPoint) { p.X = x })
				}
			}
			entryY.OnChanged = func(v string) {
				if y, err := strconv.Atoi(v); err == nil {
					_ = recorder.UpdatePoint(id, func(p *ClickPoint) { p.Y = y })
				}
			}
			entryDelay.OnChanged = func(v string) {
				if d, err := strconv.ParseFloat(v, 64); err == nil {
					_ = recorder.UpdatePoint(id, func(p *ClickPoint) { p.Delay = d })
				}
			}
			btnDelete.OnTapped = func() {
				dialog.ShowConfirm(
					"Confirm Delete",
					fmt.Sprintf("Are you sure you want to delete point %d?", id+1),
					func(confirm bool) {
						if !confirm {
							return
						}
						if err := recorder.DeletePoint(id); err != nil {
							updateLabel(statusLabel, err.Error())
							return
						}
						updateLabel(statusLabel, fmt.Sprintf("Deleted point %d", id+1))
						refreshUIList()
					},
					w,
				)
			}
		},
	)

	btnReload := widget.NewButton("Reload", func() {
		recorder.Load()
		refreshUIList()
		updateLabel(statusLabel, fmt.Sprintf("Reloaded %d points from CSV", recorder.Len()))
	})

	topBox := container.NewVBox(
		statusLabel,
		btnReload,
	)

	w.SetContent(container.NewBorder(topBox, nil, nil, nil, pointsList))
	go listenHotkeys(recorder, statusLabel, refreshUIList)
	w.ShowAndRun()
}

func togglePlayStop(label *widget.Label, recorder *Recorder) {
	playMu.Lock()
	if isPlaying {
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

func playClicks(label *widget.Label, recorder *Recorder, stopCh chan struct{}) {
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
		select {
		case <-stopCh:
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

		timer := time.NewTimer(time.Duration(delaySec * float64(time.Second)))
		select {
		case <-stopCh:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			updateLabel(label, "Stopped by user")
			return
		case <-timer.C:
		}
	}
	updateLabel(label, "Done")
}

func updateLabel(label *widget.Label, text string) {
	fyne.Do(func() { label.SetText(text) })
}
