# MiniES — Distributed Log Aggregator & Search Engine

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

MiniES is a self-contained, single-binary log ingestion and search system built in Go. It solves two operational failure modes of naive logging — **data loss under write bursts** (WAL-backed, memory-bounded) and **unusably slow search at scale** (inverted index + KMP/Bitap string matching).

> *"When a microservice fleet breaks at 2 AM, the bottleneck isn't fixing the bug — it's finding the one log line that explains it, across gigabytes of text on dozens of hosts. MiniES finds it in under half a second."*

---

## Table of Contents

- [Overview](#overview)
- [Core Problem](#core-problem)
- [Features](#features)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [API Reference](#api-reference)
- [CLI Flags](#cli-flags)
- [Search Query Syntax](#search-query-syntax)
- [Performance](#performance)
- [Build & Test](#build--test)
- [Troubleshooting](#troubleshooting)
- [Project Structure](#project-structure)
- [License](#license)

---

## Overview

MiniES ingests structured log records from multiple concurrent producers, indexes them in near-real-time, spills overflow data to disk without blocking writers, and answers **boolean/wildcard/fuzzy** queries in sub-second time — even as the corpus grows past available RAM.

### Core Problem

| Failure Mode | Root Cause | MiniES Solution |
|---|---|---|
| Data loss under burst load | No backpressure; synchronous write path blocks | Bounded channels + HTTP 429 backpressure |
| Search too slow at scale | No index — every query re-scans the full corpus | Inverted index (O(1)/O(len) lookup) |
| System crash under sustained load | Entire dataset assumed to fit in RAM | Overflow to immutable sorted segments |
| Poor UX for imprecise queries | Exact-match-only search | Bitap fuzzy + trie wildcard matching |

---

## Features

- **Real-time ingestion** via HTTP API with bounded channel backpressure (HTTP 429 when overwhelmed)
- **WAL durability** — zero data loss on restart; unflushed records recovered via WAL replay
- **In-memory inverted index** with configurable sharding (hash-partitioned)
- **Trie-based wildcard search** for prefix queries (e.g. `auth*` matches `authenticate`, `authorize`)
- **Bitap fuzzy search** for typo-tolerant matching (e.g. `tiemout~1` finds `timeout`)
- **KMP exact string matching** for precise substring search
- **Boolean query parsing** — AND, OR, NOT with set intersection/union/difference
- **Background compaction** — k-way merge of sorted segments to bound storage
- **Graceful shutdown** — drains in-flight channel, flushes buffer, closes WAL cleanly
- **In-process metrics** — `/stats` JSON endpoint for ingestion rate, index size, latency

---

## Architecture

```
Producers → HTTP API → Bounded Channel → Workers → In-Memory Index (sharded)
                                          ↓              ↓
                                     WAL (durability)  Segment Flush (sorted)
                                                       ↓
                                                 Background Compaction (k-way merge)
                                                       ↓
                                                 Disk Segments

Query Engine ← In-Memory Index + On-Disk Segments
  ├── Boolean parser (AST evaluation)
  ├── Wildcard resolver (trie prefix walk)
  ├── KMP exact matcher
  └── Bitap fuzzy matcher (edit-distance ≤ k)
```

### Data Flow

1. Producer sends a log line to the Ingestion API
2. API validates and hands the record to the async event loop via a bounded channel
3. Record is appended to the WAL (durability) and tokenized
4. Tokens are routed to the correct in-memory shard (hash-partitioned)
5. If a shard's buffer exceeds the RAM threshold, it is sorted and flushed to disk; ingestion continues into a fresh buffer
6. A background goroutine periodically k-way merges disk segments
7. On query: the engine fans out across in-memory shards and relevant disk segments concurrently, applies KMP/Bitap matching, merges and ranks results

### Record Lifecycle

```
[Received] → [WAL-written] → [Indexed (in-memory)] → [Flushed (segment)] → [Merged (compacted)]
```

A record is **durable** once WAL-written (survives crash), **searchable** once indexed, and **compacted** once merged into a larger segment.

---

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
# Exact keyword search
curl "localhost:8080/search?q=error+AND+timeout"

# Fuzzy search (typo-tolerant, edit-distance 1)
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

---

## API Reference

### `POST /ingest`

Ingest a log record.

**Request Body:**
```json
{
  "timestamp": 1730000000000,
  "service": "payments",
  "level": "ERROR",
  "message": "connection timeout to db after 3 retries",
  "source_file": "payments-7.log",
  "line_no": 4821
}
```

**Responses:**
- `200` — Record acknowledged
- `400` — Malformed log line
- `429` — Channel full (backpressure); includes `Retry-After` header

### `GET /search?q=<query>`

Search ingested logs with boolean/wildcard/fuzzy syntax.

**Response:**
```json
{
  "query": "error AND timeout",
  "took_ms": 42,
  "total_matches": 3,
  "results": [
    {
      "service": "payments",
      "level": "ERROR",
      "message": "connection timeout to db after 3 retries",
      "source_file": "payments-7.log",
      "line_no": 4821,
      "timestamp": 1730000000000,
      "score": 2.1
    }
  ]
}
```

### `GET /stats`

Returns current system metrics (ingestion rate, index size, segment count, p95 latency, WAL size, uptime).

### `GET /health`

Health check endpoint.

---

## Search Query Syntax

| Syntax | Example | Description |
|--------|---------|-------------|
| Exact keyword | `error AND timeout` | Boolean AND (set intersection) |
| OR | `timeout OR time*ut` | Boolean OR (set union) |
| NOT | `error AND payments NOT retry` | Boolean NOT (set difference) |
| Wildcard | `auth*` | Prefix match via trie walk |
| Fuzzy | `tiemout~1` | Edit-distance ≤ 1 via Bitap |
| Combined | `error AND (timeout OR time*ut~1)` | Full boolean expression with nested grouping |

---

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--listen` | `:8080` | Listen address |
| `--shards` | `8` | Number of index shards |
| `--ram-threshold` | `64MB` | Memory threshold for segment flush |
| `--segment-dir` | `./data/segments` | Directory for on-disk segment files |
| `--wal-path` | `./data/wal` | Path for Write-Ahead Log |

Environment variables are also supported (e.g. `MINIES_LISTEN`, `MINIES_SHARDS`, `MINIES_RAM_THRESHOLD`, `MINIES_SEGMENT_DIR`, `MINIES_WAL_PATH`).

---

## Performance

| Corpus Size | p95 Query Latency | RAM Footprint | Notes |
|---|---|---|---|
| 100 MB | <100ms | <50 MB | Fully in-memory scenario |
| 1 GB | <300ms | ~100–150 MB | First overflow-to-disk triggers |
| 5 GB | <800ms | ~150–250 MB | Cross-segment fan-out dominates |
| 50 GB | <1.5s | Bounded per shard | Requires sharding (stretch goal) |

- Inverted-index lookup demonstrates ≥100x speedup over linear scan at 1GB corpus size
- 0 dropped log lines during sustained 10,000 lines/sec, 10-minute load test
- Fuzzy search (edit-distance ≤1) achieves ≥90% recall against labeled typo test set

---

## Build & Test

```bash
go build -o bin/minies ./cmd/minies
go vet ./...
go test ./...
```

---

## Troubleshooting

| Symptom | Likely Cause | Resolution |
|---|---|---|
| HTTP 429 on ingest | Ingestion outpacing indexing workers | Increase `--shards`, check `/stats` |
| Search returns stale data | Data still in WAL, not yet flushed | Confirm ingestion-to-searchable lag <1s |
| Segment count growing unbounded | Background merger not running | Run `minies compact --force` |
| Process restart lost data | WAL not enabled or replay failed | Verify `--wal-path` is set and writable |
| Fuzzy query very slow | `k` too large relative to token length | Cap `k` server-side (max 3) |

---

## Project Structure

```
minies/
├── cmd/minies/main.go          Entry point & CLI flag parsing
├── internal/
│   ├── ingest/                 HTTP server & handlers
│   ├── pipeline/               Async event loop + backpressure
│   ├── index/                  Inverted index, trie, shards
│   ├── storage/                WAL, segments, flush, merge
│   ├── search/                 Query parser, KMP, Bitap, executor
│   ├── tokenizer/              Tokenization (lowercase, word splitting)
│   └── metrics/                In-process counters & gauges
├── pkg/logrecord/              Log record model and validation
├── go.mod                      Module definition
├── AGENTS.md                   Project agent configuration
└── LICENSE
```

### Subsystems

| Package | Description |
|---------|-------------|
| `pkg/logrecord` | Log record model and validation |
| `internal/tokenizer` | Tokenization (lowercase, word splitting) |
| `internal/index` | Sharded in-memory inverted index + trie |
| `internal/storage` | WAL, sorted segments, flush, compaction |
| `internal/search` | Query parser, KMP, Bitap, boolean executor |
| `internal/pipeline` | Async event loop with backpressure |
| `internal/ingest` | HTTP server for ingest/search/stats |
| `internal/metrics` | In-process counters and gauges |

---

## License

[MIT](LICENSE)
