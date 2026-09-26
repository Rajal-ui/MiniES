package index

import "sync"

type Shard struct {
	id     int
	data   map[string][]Posting
	mu     sync.RWMutex
	buffer map[string][]Posting
}

func NewShard(id int) *Shard {
	return &Shard{
		id:     id,
		data:   make(map[string][]Posting),
		buffer: make(map[string][]Posting),
	}
}

func (s *Shard) Add(token string, posting Posting) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buffer[token] = append(s.buffer[token], posting)
}

func (s *Shard) Flush() map[string][]Posting {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := s.buffer
	s.buffer = make(map[string][]Posting)
	for token, postings := range result {
		s.data[token] = append(s.data[token], postings...)
	}
	return result
}

func (s *Shard) Get(token string) []Posting {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data := s.data[token]
	buf := s.buffer[token]
	result := make([]Posting, 0, len(data)+len(buf))
	result = append(result, data...)
	result = append(result, buf...)
	return result
}

// RemapPending rewrites every unflushed posting (SegmentID == -1) to segID.
func (s *Shard) RemapPending(segID int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range []map[string][]Posting{s.data, s.buffer} {
		for _, postings := range m {
			for i := range postings {
				if postings[i].SegmentID == -1 {
					postings[i].SegmentID = segID
				}
			}
		}
	}
}

func (s *Shard) GetAllTokens() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tokens := make([]string, 0, len(s.data)+len(s.buffer))
	for t := range s.data {
		tokens = append(tokens, t)
	}
	for t := range s.buffer {
		tokens = append(tokens, t)
	}
	return tokens
}

func (s *Shard) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data) + len(s.buffer)
}

func (s *Shard) Data() map[string][]Posting {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string][]Posting)
	for k, v := range s.data {
		result[k] = append([]Posting(nil), v...)
	}
	return result
}
