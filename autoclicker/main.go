package main

import (
	"fmt"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/go-vgo/robotgo"
)

func main() {
	a := app.New()
	w := a.NewWindow("Auto Clicker " + version)
	w.Resize(fyne.NewSize(100, 300))

	statusLabel := widget.NewLabel("Press F2 to start recording")
	recorder := NewRecorder()

	var displayPoints []ClickPoint
	var pointsList *widget.List
	var refreshUIList func()

	pointsList = widget.NewList(
		func() int {
			return len(displayPoints)
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

			pt := displayPoints[id]

			labelNum.SetText(fmt.Sprintf("%d", id+1))
			entryX.SetText(fmt.Sprintf("%d", pt.X))
			entryY.SetText(fmt.Sprintf("%d", pt.Y))
			entryDelay.SetText(fmt.Sprintf("%.1f", pt.Delay))

			entryX.OnChanged = func(val string) {
				if x, err := strconv.Atoi(val); err == nil {
					displayPoints[id].X = x
					_ = savePointsToCSV(displayPoints)
				}
			}

			entryY.OnChanged = func(val string) {
				if y, err := strconv.Atoi(val); err == nil {
					displayPoints[id].Y = y
					_ = savePointsToCSV(displayPoints)
				}
			}

			entryDelay.OnChanged = func(val string) {
				if d, err := strconv.ParseFloat(val, 64); err == nil {
					displayPoints[id].Delay = d
					_ = savePointsToCSV(displayPoints)
				}
			}

			rowIndex := id
			btnDelete.OnTapped = func() {
				if recorder.IsRecording() {
					updateLabel(statusLabel, "Cannot delete while recording")
					return
				}
				if rowIndex >= 0 && rowIndex < len(displayPoints) {
					displayPoints = append(displayPoints[:rowIndex], displayPoints[rowIndex+1:]...)
					if err := savePointsToCSV(displayPoints); err != nil {
						updateLabel(statusLabel, fmt.Sprintf("Error saving file: %v", err))
						return
					}
					updateLabel(statusLabel, "Deleted point")
					refreshUIList()
				}
			}
		},
	)

	refreshUIList = func() {
		fyne.Do(func() {
			if recorder.IsRecording() {
				displayPoints = recorder.GetRecorded()
			} else {
				displayPoints = loadPointsFromCSV()
			}
			pointsList.Refresh()
		})
	}

	displayPoints = loadPointsFromCSV()

	btnPlay := widget.NewButton("Play", func() {
		go playClicks(statusLabel)
	})
	btnReload := widget.NewButton("Reload", func() {
		refreshUIList()
		updateLabel(statusLabel, fmt.Sprintf("Reloaded %d points from CSV", len(displayPoints)))
	})

	tableHeader := container.NewGridWithColumns(5,
		widget.NewLabelWithStyle("#", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("X", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Y", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Delay (s)", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Action", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
	)

	topBox := container.NewVBox(
		statusLabel,
		container.NewGridWithColumns(2, btnPlay, btnReload),
		tableHeader,
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

func playClicks(label *widget.Label) {
	points := loadPointsFromCSV()
	if len(points) == 0 {
		updateLabel(label, "Warning: No points in file or file not found")
		return
	}
	updateLabel(label, "Play")
	for _, pt := range points {
		robotgo.Move(pt.X, pt.Y)
		robotgo.Click(pt.Button, false)
		delaySec := pt.Delay
		if delaySec <= 0 {
			delaySec = defaultDelaySec
		}
		time.Sleep(time.Duration(delaySec * float64(time.Second)))
	}
	updateLabel(label, "Done")
}

func updateLabel(label *widget.Label, text string) {
	fyne.Do(func() {
		label.SetText(text)
	})
}
