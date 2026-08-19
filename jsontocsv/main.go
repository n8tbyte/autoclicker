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

	// 1. อ่านข้อมูลจากไฟล์ JSON
	fileData, err := os.ReadFile(jsonFilename)
	if err != nil {
		fmt.Println("Error reading JSON file:", err)
		return
	}

	// 2. แปลง JSON เป็น Struct Slice
	var actions []ClickAction
	err = json.Unmarshal(fileData, &actions)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	// 3. สร้างไฟล์ CSV
	file, err := os.Create(csvFilename)
	if err != nil {
		fmt.Println("Error creating CSV file:", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 4. เขียน Header
	writer.Write([]string{"x", "y", "button", "delay"})

	// 5. เขียนข้อมูลแต่ละแถว
	for _, a := range actions {
		row := []string{
			strconv.Itoa(a.X),
			strconv.Itoa(a.Y),
			a.Button,
			fmt.Sprintf("%g", a.Delay), // ใช้ %g เพื่อจัดการทศนิยมให้ดูสะอาดตา
		}
		writer.Write(row)
	}

	fmt.Printf("Successfully converted %s to %s\n", jsonFilename, csvFilename)
}
