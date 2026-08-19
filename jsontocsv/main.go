package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
)

// ปรับ struct ให้ตรงกับ JSON
type StreamItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func main() {
	jsonFilename := "autoclicker-db.json"
	csvFilename := "autoclicker-db.csv"

	fileData, err := os.ReadFile(jsonFilename)
	if err != nil {
		fmt.Println("Error reading JSON file:", err)
		return
	}

	var items []StreamItem // ใช้ slice ของ struct ใหม่
	err = json.Unmarshal(fileData, &items)
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

	// เขียน Header
	writer.Write([]string{"key", "value"})

	// วนลูปเขียนข้อมูล
	for _, item := range items {
		row := []string{
			item.Key,
			item.Value,
		}
		writer.Write(row)
	}

	fmt.Printf("Successfully converted %s to %s\n", jsonFilename, csvFilename)
}
