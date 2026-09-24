package pipeline

import (
	"sync"
	"time"

	"github.com/rajal-ui/minies/internal/index"
	"github.com/rajal-ui/minies/internal/storage"
	"github.com/rajal-ui/minies/internal/tokenizer"
	"github.com/rajal-ui/minies/pkg/logrecord"
)

type EventLoop struct {
	inbound  chan *logrecord.LogRecord
	workers  int
	index    *index.InvertedIndex
	trie     *index.Trie
	flusher  *storage.Flusher
	segments *storage.SegmentManager
	wal      *storage.WAL
	shutdown chan struct{}
	wg       sync.WaitGroup
	mu       sync.RWMutex
	running  bool
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
		inbound:  make(chan *logrecord.LogRecord, 10000),
		workers:  workers,
		index:    idx,
		trie:     trie,
		flusher:  flusher,
		segments: segments,
		wal:      wal,
		shutdown: make(chan struct{}),
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
	if err := el.wal.Append([]byte(record.Message)); err != nil {
		return
	}
	el.wal.Flush()
	tokens := record.Tokenize()
	for _, token := range tokens {
		normalized := tokenizer.NormalizeToken(token)
		el.index.Add(normalized, index.Posting{})
		el.trie.Insert(normalized)
	}
}

func (el *EventLoop) flusherLoop() {
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
	for _, shard := range el.index.GetShards() {
		data := shard.Flush()
		if len(data) > 0 {
			var records [][]byte
			for token := range data {
				records = append(records, []byte(token))
			}
			if len(records) > 0 {
				el.flusher.Flush(records)
			}
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
	close(el.shutdown)
	el.flushBuffers()
	el.wg.Wait()
	el.wal.Close()
	el.mu.Lock()
	el.running = false
	el.mu.Unlock()
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
