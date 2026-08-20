package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
)

const (
	fileName        = "autoclicker-db.csv"
	defaultDelaySec = 1.0
)

func savePointsToCSV(points []ClickPoint) error {
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	w := csv.NewWriter(file)
	defer w.Flush()

	// เอาการเขียน header w.Write([]string{"x", "y", "button", "delay"}) ออกแล้ว

	for _, pt := range points {
		rec := []string{
			strconv.Itoa(pt.X),
			strconv.Itoa(pt.Y),
			pt.Button,
			strconv.FormatFloat(pt.Delay, 'f', -1, 64),
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return nil
}

func loadPointsFromCSV() []ClickPoint {
	file, err := os.Open(fileName)
	if err != nil {
		return nil
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	// เปลี่ยนเช็คความยาวจาก <= 1 เป็น == 0
	if err != nil || len(records) == 0 {
		return nil
	}

	var points []ClickPoint
	// เปลี่ยนจาก records[1:] มาวนลูปอ่าน records ทั้งหมด
	for _, rec := range records {
		if len(rec) < 4 {
			continue
		}
		x, errX := strconv.Atoi(rec[0])
		y, errY := strconv.Atoi(rec[1])
		delay, errD := strconv.ParseFloat(rec[3], 64)

		if errX == nil && errY == nil && errD == nil {
			points = append(points, ClickPoint{X: x, Y: y, Button: rec[2], Delay: delay})
		}
	}
	return points
}
