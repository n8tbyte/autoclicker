package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
)

const (
	fileName        = "autoclicker-store.csv"
	defaultDelaySec = 1.0
)

func savePointsToCSV(points []ClickPoint) error {
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{"x", "y", "button", "delay"}); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	for _, pt := range points {
		record := []string{
			strconv.Itoa(pt.X),
			strconv.Itoa(pt.Y),
			pt.Button,
			strconv.FormatFloat(pt.Delay, 'f', -1, 64),
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write record: %w", err)
		}
	}
	return nil
}

func loadPointsFromCSV() []ClickPoint {
	file, err := os.Open(fileName)
	if err != nil {
		return []ClickPoint{}
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil || len(records) <= 1 {
		return []ClickPoint{}
	}

	var points []ClickPoint

	for _, record := range records[1:] {
		if len(record) < 4 {
			continue
		}
		x, errX := strconv.Atoi(record[0])
		y, errY := strconv.Atoi(record[1])
		button := record[2]
		delay, errD := strconv.ParseFloat(record[3], 64)

		if errX != nil || errY != nil || errD != nil {
			continue
		}

		points = append(points, ClickPoint{
			X:      x,
			Y:      y,
			Button: button,
			Delay:  delay,
		})
	}

	return points
}
