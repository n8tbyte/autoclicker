package main

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	fileName        = "autoclicker-store.json"
	defaultDelaySec = 1.0
)

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