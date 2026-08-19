package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

// แก้ไขฟิลด์ Delay จาก int เป็น float64 เพื่อรองรับเลขทศนิยม
type ClickAction struct {
	X      int     `json:"x"`
	Y      int     `json:"y"`
	Button string  `json:"button"`
	Delay  float64 `json:"delay"`
}

func main() {
	filename := "autoclicker-store.json"

	// 1. อ่านข้อมูลจากไฟล์ JSON เดิม
	fileData, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	// 2. แปลงข้อมูล JSON เป็น Go Slice of Struct
	var actions []ClickAction
	err = json.Unmarshal(fileData, &actions)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	// 3. จัด Format ให้ขึ้นบรรทัดใหม่ทุกๆ 10 Object
	formattedJSON := formatJSONPerLine(actions, 10)

	// 4. เขียนทับ (Overwrite) ข้อมูลลงไปที่ไฟล์เดิม
	err = os.WriteFile(filename, []byte(formattedJSON), 0644)
	if err != nil {
		fmt.Println("Error writing file:", err)
		return
	}

	fmt.Println("Successfully updated and overwritten", filename)
}

// ฟังก์ชันจัดรูปแบบ JSON ให้แสดงผลแถวละ 10 รายการ
func formatJSONPerLine(actions []ClickAction, itemsPerLine int) string {
	var buf bytes.Buffer
	buf.WriteString("[\n")

	for i, action := range actions {
		// แปลง Object แต่ละตัวเป็น JSON string บรรทัดเดียว
		itemBytes, _ := json.Marshal(action)

		if i > 0 {
			buf.WriteString(", ")
		}

		// ถ้าครบ 10 ตัว ให้ขึ้นบรรทัดใหม่
		if i > 0 && i%itemsPerLine == 0 {
			buf.WriteString("\n")
		}

		buf.Write(itemBytes)
	}

	buf.WriteString("\n]")
	return buf.String()
}
