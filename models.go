package main

type ClickPoint struct {
	X      int     `json:"x"`
	Y      int     `json:"y"`
	Button string  `json:"button"`
	Delay  float64 `json:"delay"`
}