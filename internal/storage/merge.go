package storage

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
	"sort"
	"sync"
)

type SegmentManager struct {
	dir      string
	segments []*Segment
	mu       sync.RWMutex
	nextID   int
}

func NewSegmentManager(dir string) *SegmentManager {
	os.MkdirAll(dir, 0755)
	sm := &SegmentManager{dir: dir}
	sm.loadSegments()
	return sm
}

func (sm *SegmentManager) loadSegments() {
	entries, _ := os.ReadDir(sm.dir)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if len(name) >= 8 && name[:8] == "segment_" {
			id := sm.nextID
			if n, err := fmt.Sscanf(name, "segment_%d.bin", &id); err == nil && n == 1 {
				if id >= sm.nextID {
					sm.nextID = id + 1
				}
			}
			sm.segments = append(sm.segments, &Segment{ID: id, Path: sm.dir + "/" + name})
		}
	}
	sort.Slice(sm.segments, func(i, j int) bool {
		return sm.segments[i].ID < sm.segments[j].ID
	})
}

func (sm *SegmentManager) AddSegment(seg *Segment) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if seg.ID >= sm.nextID {
		sm.nextID = seg.ID + 1
	}
	sm.segments = append(sm.segments, seg)
}

func (sm *SegmentManager) GetSegments() []*Segment {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	result := make([]*Segment, len(sm.segments))
	copy(result, sm.segments)
	return result
}

func (sm *SegmentManager) SegmentCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.segments)
}

func (sm *SegmentManager) Dir() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.dir
}

// NextID returns the next segment ID to allocate.
func (sm *SegmentManager) NextID() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.nextID
}

func (sm *SegmentManager) Compact() ([]*Segment, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if len(sm.segments) < 2 {
		return nil, nil
	}
	allRecords := [][]byte{}
	for _, seg := range sm.segments {
		records, err := ReadSegment(seg.Path)
		if err != nil {
			continue
		}
		allRecords = append(allRecords, records...)
	}
	if len(allRecords) == 0 {
		return nil, nil
	}
	sort.Slice(allRecords, func(i, j int) bool {
		return string(allRecords[i]) < string(allRecords[j])
	})
	newSeg := &Segment{
		ID:    sm.nextID,
		Path:  fmt.Sprintf("%s/segment_%06d.bin", sm.dir, sm.nextID),
		Count: len(allRecords),
	}
	sm.nextID++
	if err := WriteSegmentFile(newSeg.Path, allRecords); err != nil {
		return nil, err
	}
	for _, seg := range sm.segments {
		os.Remove(seg.Path)
	}
	sm.segments = []*Segment{newSeg}
	return []*Segment{newSeg}, nil
}

func WriteSegmentFile(path string, records [][]byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	writer := bufio.NewWriter(f)
	for _, rec := range records {
		if err := binary.Write(writer, binary.LittleEndian, uint32(len(rec))); err != nil {
			return err
		}
		if _, err := writer.Write(rec); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func (sm *SegmentManager) Close() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for _, seg := range sm.segments {
		os.Remove(seg.Path)
	}
}
