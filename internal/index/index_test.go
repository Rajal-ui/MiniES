package index

import "testing"

func TestInvertedIndexAddAndGet(t *testing.T) {
	idx := NewInvertedIndex(4)
	idx.Add("hello", Posting{SegmentID: 0, Offset: 0})
	idx.Add("hello", Posting{SegmentID: 0, Offset: 1})
	idx.Add("world", Posting{SegmentID: 1, Offset: 0})

	postings := idx.Get("hello")
	if len(postings) != 2 {
		t.Errorf("Expected 2 postings for 'hello', got %d", len(postings))
	}

	postings = idx.Get("world")
	if len(postings) != 1 {
		t.Errorf("Expected 1 posting for 'world', got %d", len(postings))
	}

	postings = idx.Get("nonexistent")
	if len(postings) != 0 {
		t.Errorf("Expected 0 postings for 'nonexistent', got %d", len(postings))
	}
}

func TestInvertedIndexSize(t *testing.T) {
	idx := NewInvertedIndex(4)
	if idx.Size() != 0 {
		t.Errorf("Expected size 0, got %d", idx.Size())
	}
	idx.Add("hello", Posting{})
	if idx.Size() != 1 {
		t.Errorf("Expected size 1, got %d", idx.Size())
	}
}

func TestInvertedIndexShardCount(t *testing.T) {
	idx := NewInvertedIndex(8)
	if idx.NumShards() != 8 {
		t.Errorf("Expected 8 shards, got %d", idx.NumShards())
	}
}

func TestShardAddAndFlush(t *testing.T) {
	shard := NewShard(0)
	shard.Add("hello", Posting{SegmentID: 0, Offset: 0})
	shard.Add("hello", Posting{SegmentID: 0, Offset: 1})

	data := shard.Flush()
	if len(data) != 1 {
		t.Errorf("Expected 1 token after flush, got %d", len(data))
	}
	if len(data["hello"]) != 2 {
		t.Errorf("Expected 2 postings for 'hello', got %d", len(data["hello"]))
	}

	postings := shard.Get("hello")
	if len(postings) != 2 {
		t.Errorf("Expected 2 postings after flush, got %d", len(postings))
	}
}

func TestTrieInsertAndSearch(t *testing.T) {
	trie := NewTrie()
	trie.Insert("hello")
	trie.Insert("world")
	trie.Insert("help")

	results := trie.SearchPrefix("hel")
	if len(results) != 2 {
		t.Errorf("Expected 2 results for prefix 'hel', got %d", len(results))
	}

	results = trie.SearchPrefix("wor")
	if len(results) != 1 {
		t.Errorf("Expected 1 result for prefix 'wor', got %d", len(results))
	}

	if !trie.Contains("hello") {
		t.Error("Trie should contain 'hello'")
	}
	if trie.Contains("hell") {
		t.Error("Trie should not contain 'hell'")
	}
}
