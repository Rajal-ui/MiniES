package pipeline

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/rajal-ui/minies/internal/index"
	"github.com/rajal-ui/minies/internal/storage"
	"github.com/rajal-ui/minies/internal/tokenizer"
	"github.com/rajal-ui/minies/pkg/logrecord"
)

type EventLoop struct {
	inbound        chan *logrecord.LogRecord
	workers        int
	index          *index.InvertedIndex
	trie           *index.Trie
	flusher        *storage.Flusher
	segments       *storage.SegmentManager
	wal            *storage.WAL
	shutdown       chan struct{}
	wg             sync.WaitGroup
	mu             sync.RWMutex
	stopOnce       sync.Once
	running        bool
	pendingRecords [][]byte
}

func NewEventLoop(
	workers int,
	idx *index.InvertedIndex,
	trie *index.Trie,
	flusher *storage.Flusher,
	segments *storage.SegmentManager,
	wal *storage.WAL,
) *EventLoop {
	if workers < 1 {
		workers = 1
	}
	return &EventLoop{
		inbound:        make(chan *logrecord.LogRecord, 10000),
		workers:        workers,
		index:          idx,
		trie:           trie,
		flusher:        flusher,
		segments:       segments,
		wal:            wal,
		shutdown:       make(chan struct{}),
		pendingRecords: make([][]byte, 0, 1000),
	}
}

func (el *EventLoop) Start() {
	el.mu.Lock()
	el.running = true
	el.mu.Unlock()
	for i := 0; i < el.workers; i++ {
		el.wg.Add(1)
		go el.worker(i)
	}
	el.wg.Add(1)
	go el.flusherLoop()
}

func (el *EventLoop) worker(id int) {
	defer el.wg.Done()
	for {
		select {
		case record := <-el.inbound:
			if record == nil {
				return
			}
			el.processRecord(record)
		case <-el.shutdown:
			return
		}
	}
}

func (el *EventLoop) processRecord(record *logrecord.LogRecord) {
	if err := record.Validate(); err != nil {
		return
	}
	recordBytes, err := json.Marshal(record)
	if err != nil {
		return
	}
	if err := el.wal.Append(recordBytes); err != nil {
		return
	}
	el.wal.Flush()

	// Store the full record for later flushing
	el.mu.Lock()
	recordIndex := len(el.pendingRecords)
	el.pendingRecords = append(el.pendingRecords, recordBytes)
	el.mu.Unlock()

	tokens := record.Tokenize()
	for _, token := range tokens {
		normalized := tokenizer.NormalizeToken(token)
		// Use recordIndex as offset in posting (will be updated with segment ID during flush)
		el.index.Add(normalized, index.Posting{SegmentID: -1, Offset: recordIndex})
		el.trie.Insert(normalized)
	}
}

func (el *EventLoop) flusherLoop() {
	defer el.wg.Done()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			el.flushBuffers()
		case <-el.shutdown:
			el.flushBuffers()
			return
		}
	}
}

func (el *EventLoop) flushBuffers() {
	el.mu.Lock()
	defer el.mu.Unlock()
	if len(el.pendingRecords) == 0 {
		return
	}
	// Hold el.mu for the whole flush so no record can add postings with
	// SegmentID == -1 while we are remapping them.
	recordsToFlush := el.pendingRecords
	el.pendingRecords = make([][]byte, 0, 1000)

	segID := el.segments.NextID()
	seg, err := storage.WriteSegment(segID, el.segments.Dir(), recordsToFlush)
	if err != nil {
		// Put records back so they are retried on the next tick.
		el.pendingRecords = append(recordsToFlush, el.pendingRecords...)
		return
	}
	el.segments.AddSegment(seg)

	// Point all unflushed postings at the segment that now owns them.
	for _, shard := range el.index.GetShards() {
		shard.RemapPending(segID)
	}
}

// Recover re-adds records that were written to the WAL but never made it
// into a segment, so they survive a restart. It must be called before Start.
func (el *EventLoop) Recover(records [][]byte) {
	el.mu.Lock()
	defer el.mu.Unlock()
	for _, raw := range records {
		var rec logrecord.LogRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			continue
		}
		if err := rec.Validate(); err != nil {
			continue
		}
		recordIndex := len(el.pendingRecords)
		el.pendingRecords = append(el.pendingRecords, raw)
		for _, token := range rec.Tokenize() {
			normalized := tokenizer.NormalizeToken(token)
			el.index.Add(normalized, index.Posting{SegmentID: -1, Offset: recordIndex})
			el.trie.Insert(normalized)
		}
	}
}

func (el *EventLoop) Ingest(record *logrecord.LogRecord) error {
	select {
	case el.inbound <- record:
		return nil
	case <-el.shutdown:
		return ErrShutdown
	}
}

func (el *EventLoop) Stop() {
	el.stopOnce.Do(func() {
		close(el.shutdown)
		el.wg.Wait()
		el.flushBuffers()
		el.wal.Close()
		el.mu.Lock()
		el.running = false
		el.mu.Unlock()
	})
}

func (el *EventLoop) Stats() map[string]int {
	return map[string]int{
		"channel_capacity": cap(el.inbound),
		"channel_len":      len(el.inbound),
		"workers":          el.workers,
		"index_size":       el.index.Size(),
		"shard_count":      el.index.NumShards(),
	}
}

var ErrShutdown = &PipelineError{"pipeline shutdown"}

type PipelineError struct {
	msg string
}

func (e *PipelineError) Error() string { return e.msg }
