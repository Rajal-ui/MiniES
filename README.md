# MiniES — Distributed Log Aggregator & Search Engine

## Overview

MiniES is a Go-based single-binary log ingestion and search system. It solves two operational failure modes of naive logging: **data loss under write bursts** (WAL-backed, memory-bounded) and **unusably slow search at scale** (inverted index + KMP/Bitap).

## Quick Start

### Build

```bash
go build -o bin/minies ./cmd/minies
```

### Run

```bash
./bin/minies --listen=:8080 --shards=8 --ram-threshold=64MB --segment-dir=./data/segments --wal-path=./data/wal
```

### Ingest Logs

```bash
curl -X POST localhost:8080/ingest -d '{"service":"payments","level":"ERROR","message":"connection timeout to db"}'
```

### Search

```bash
# Exact search
curl "localhost:8080/search?q=error+AND+timeout"

# Fuzzy search (typo-tolerant)
curl "localhost:8080/search?q=tiemout~1"

# Wildcard search
curl "localhost:8080/search?q=auth*"

# Boolean combination
curl "localhost:8080/search?q=error+AND+(timeout+OR+time*ut~1)"
```

### Stats

```bash
curl localhost:8080/stats
```

## Architecture

```
Producers → HTTP API → Bounded Channel → Workers → In-Memory Index
                                    ↓
                              WAL (durability)
                                    ↓
                              Segment Flush (sorted)
                                    ↓
                              Background Compaction (k-way merge)
```

## Subsystems

| Package | Description |
|---------|-------------|
| `pkg/logrecord` | Log record model and validation |
| `internal/tokenizer` | Tokenization (lowercase, word splitting) |
| `internal/index` | Sharded in-memory inverted index + trie |
| `internal/storage` | WAL, sorted segments, compaction |
| `internal/search` | Query parser, KMP, Bitap, boolean logic |
| `internal/pipeline` | Async event loop with backpressure |
| `internal/ingest` | HTTP server for ingest/search/stats |
| `internal/metrics` | In-process counters and gauges |

## Project Structure

```
minies/
├── cmd/minies/main.go
├── internal/
│   ├── ingest/       HTTP server & handlers
│   ├── pipeline/     Async event loop + backpressure
│   ├── index/        Inverted index, trie, shards
│   ├── storage/      WAL, segments, flush, merge
│   ├── search/       Parser, KMP, Bitap, executor
│   ├── tokenizer/    Tokenization
│   └── metrics/      In-process metrics
├── pkg/logrecord/    Log record model
├── test/             Test suites
└── AGENTS.md         Project agent config
```

## Features

- **Real-time ingestion** via HTTP API with bounded channel backpressure
- **WAL durability** — zero data loss on restart
- **In-memory inverted index** with configurable sharding
- **Trie-based wildcard search** for prefix queries
- **Bitap fuzzy search** for typo-tolerant matching
- **KMP exact string matching**
- **Boolean query parsing** (AND, OR, NOT)
- **Background compaction** (k-way merge of sorted segments)
- **Graceful shutdown** with channel drain and WAL close

## Build & Test

```bash
go build -o bin/minies ./cmd/minies
go vet ./...
go test ./...
```

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--listen` | `:8080` | Listen address |
| `--shards` | `8` | Number of index shards |
| `--ram-threshold` | `64MB` | Memory threshold for flush |
| `--segment-dir` | `./data/segments` | Segment file directory |
| `--wal-path` | `./data/wal` | WAL directory |

## License

MIT
