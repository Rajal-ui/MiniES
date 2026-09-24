package storage

import (
	"sort"
)

type Flusher struct {
	segments *SegmentManager
}

func NewFlusher(segments *SegmentManager) *Flusher {
	return &Flusher{segments: segments}
}

func (f *Flusher) Flush(records [][]byte) error {
	sort.Slice(records, func(i, j int) bool {
		return string(records[i]) < string(records[j])
	})
	segID := f.segments.SegmentCount()
	seg, err := WriteSegment(segID, f.segments.dir, records)
	if err != nil {
		return err
	}
	f.segments.AddSegment(seg)
	return nil
}

func (f *Flusher) Close() {}
