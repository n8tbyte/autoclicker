package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

type ClickAction struct {
	X      int     `json:"x"`
	Y      int     `json:"y"`
	Button string  `json:"button"`
	Delay  float64 `json:"delay"`
}

func main() {
	filename := "autoclicker-db.json"

	fileData, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	var actions []ClickAction
	err = json.Unmarshal(fileData, &actions)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	formattedJSON := formatJSONPerLine(actions, 10)

	err = os.WriteFile(filename, []byte(formattedJSON), 0644)
	if err != nil {
		fmt.Println("Error writing file:", err)
		return
	}

	fmt.Println("Successfully updated and overwritten", filename)
}

func formatJSONPerLine(actions []ClickAction, itemsPerLine int) string {
	var buf bytes.Buffer
	buf.WriteString("[\n")

	for i, action := range actions {

		itemBytes, _ := json.Marshal(action)

		if i > 0 {
			buf.WriteString(", ")
		}

		if i > 0 && i%itemsPerLine == 0 {
			buf.WriteString("\n")
		}

		buf.Write(itemBytes)
	}

	buf.WriteString("\n]")
	return buf.String()
}
