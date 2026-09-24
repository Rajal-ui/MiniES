package index

type Posting struct {
	SegmentID int
	Offset    int
}

type InvertedIndex struct {
	shards []*Shard
	count  int
}

func NewInvertedIndex(shardCount int) *InvertedIndex {
	if shardCount < 1 {
		shardCount = 1
	}
	shards := make([]*Shard, shardCount)
	for i := 0; i < shardCount; i++ {
		shards[i] = NewShard(i)
	}
	return &InvertedIndex{
		shards: shards,
		count:  shardCount,
	}
}

func (idx *InvertedIndex) shardIndex(token string) int {
	h := 0
	for _, c := range token {
		h = 31*h + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h % idx.count
}

func (idx *InvertedIndex) Add(token string, posting Posting) {
	shard := idx.shards[idx.shardIndex(token)]
	shard.Add(token, posting)
}

func (idx *InvertedIndex) Get(token string) []Posting {
	shard := idx.shards[idx.shardIndex(token)]
	return shard.Get(token)
}

func (idx *InvertedIndex) GetAllTokens() []string {
	var tokens []string
	for _, s := range idx.shards {
		tokens = append(tokens, s.GetAllTokens()...)
	}
	return tokens
}

func (idx *InvertedIndex) Size() int {
	total := 0
	for _, s := range idx.shards {
		total += s.Size()
	}
	return total
}

func (idx *InvertedIndex) NumShards() int { return idx.count }

func (idx *InvertedIndex) GetShards() []*Shard { return idx.shards }

func (idx *InvertedIndex) ShardSizes() []int {
	sizes := make([]int, idx.count)
	for i, s := range idx.shards {
		sizes[i] = s.Size()
	}
	return sizes
}

func (idx *InvertedIndex) Drain() map[string][]Posting {
	result := make(map[string][]Posting)
	for _, s := range idx.shards {
		for token, postings := range s.Data() {
			result[token] = append(result[token], postings...)
		}
	}
	return result
}

func (idx *InvertedIndex) Restore(data map[string][]Posting) {
	for token, postings := range data {
		for _, p := range postings {
			idx.Add(token, p)
		}
	}
}
