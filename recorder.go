package main

import (
	"fmt"
	"sync"
)

type Recorder struct {
	mu          sync.RWMutex
	isRecording bool
	recorded    []ClickPoint
}

func NewRecorder() *Recorder {
	r := &Recorder{}
	r.Load()
	return r
}

func (r *Recorder) IsRecording() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.isRecording
}

func (r *Recorder) Load() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recorded = loadPointsFromCSV()
}

func (r *Recorder) GetRecorded() []ClickPoint {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pts := make([]ClickPoint, len(r.recorded))
	copy(pts, r.recorded)
	return pts
}

func (r *Recorder) GetAt(index int) (ClickPoint, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if index < 0 || index >= len(r.recorded) {
		return ClickPoint{}, false
	}
	return r.recorded[index], true
}

func (r *Recorder) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.recorded)
}

func (r *Recorder) Start() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.isRecording = true
	r.recorded = loadPointsFromCSV()
	return len(r.recorded)
}

func (r *Recorder) Stop() ([]ClickPoint, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.isRecording = false
	err := r.saveLocked()
	return r.recorded, err
}

func (r *Recorder) AddPoint(x, y int, button string, delay float64) (int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.isRecording {
		return 0, false
	}
	r.recorded = append(r.recorded, ClickPoint{X: x, Y: y, Button: button, Delay: delay})
	return len(r.recorded), true
}

func (r *Recorder) UpdatePoint(index int, updateFn func(pt *ClickPoint)) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if index < 0 || index >= len(r.recorded) {
		return fmt.Errorf("index out of range")
	}
	updateFn(&r.recorded[index])
	return r.saveLocked()
}

func (r *Recorder) DeletePoint(index int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.isRecording {
		return fmt.Errorf("cannot delete while recording")
	}
	if index < 0 || index >= len(r.recorded) {
		return fmt.Errorf("index out of range")
	}
	r.recorded = append(r.recorded[:index], r.recorded[index+1:]...)
	return r.saveLocked()
}

func (r *Recorder) saveLocked() error {
	return savePointsToCSV(r.recorded)
}
