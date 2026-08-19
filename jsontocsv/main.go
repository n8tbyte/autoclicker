package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

type ClickAction struct {
	X      int     `json:"x"`
	Y      int     `json:"y"`
	Button string  `json:"button"`
	Delay  float64 `json:"delay"`
}

func main() {
	jsonFilename := "autoclicker-store.json"
	csvFilename := "autoclicker-store.csv"

	fileData, err := os.ReadFile(jsonFilename)
	if err != nil {
		fmt.Println("Error reading JSON file:", err)
		return
	}

	var actions []ClickAction
	err = json.Unmarshal(fileData, &actions)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	file, err := os.Create(csvFilename)
	if err != nil {
		fmt.Println("Error creating CSV file:", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"x", "y", "button", "delay"})

	for _, a := range actions {
		row := []string{
			strconv.Itoa(a.X),
			strconv.Itoa(a.Y),
			a.Button,
			fmt.Sprintf("%g", a.Delay),
		}
		writer.Write(row)
	}

	fmt.Printf("Successfully converted %s to %s\n", jsonFilename, csvFilename)
}
