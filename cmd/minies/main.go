package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/rajal-ui/minies/internal/index"
	"github.com/rajal-ui/minies/internal/ingest"
	"github.com/rajal-ui/minies/internal/pipeline"
	"github.com/rajal-ui/minies/internal/search"
	"github.com/rajal-ui/minies/internal/storage"
)

func main() {
	listen       := flag.String("listen", ":8080", "Listen address")
	shards       := flag.Int("shards", 8, "Number of index shards")
	segmentDir   := flag.String("segment-dir", "./data/segments", "Segment directory")
	walPath      := flag.String("wal-path", "./data/wal", "WAL directory")
	ramThreshold := flag.String("ram-threshold", "64MB", "RAM threshold before flushing index shards to disk (e.g. 64MB, 256MB)")
	flag.Parse()

	idx := index.NewInvertedIndex(*shards)
	trie := index.NewTrie()
	wal, err := storage.NewWAL(*walPath)
	if err != nil {
		log.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	segments := storage.NewSegmentManager(*segmentDir)
	flusher := storage.NewFlusher(segments)

	records, err := wal.Replay()
	if err != nil {
		log.Printf("WAL replay error: %v", err)
	}
	_ = records

	eventLoop := pipeline.NewEventLoop(
		*shards, idx, trie, flusher, segments, wal,
	)
	eventLoop.Start()
	defer eventLoop.Stop()

	executor := search.NewExecutor(idx, trie, segments)
	_ = executor

	go func() {
		if err := ingest.RunServer(*listen, eventLoop); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	fmt.Printf("MiniES started\n")
	fmt.Printf("  Listen:        %s\n", *listen)
	fmt.Printf("  Shards:        %d\n", *shards)
	fmt.Printf("  RAM threshold: %s\n", *ramThreshold)
	fmt.Printf("  Segments:      %s\n", *segmentDir)
	fmt.Printf("  WAL:           %s\n", *walPath)
	fmt.Printf("Press Ctrl+C to stop\n")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	eventLoop.Stop()
	fmt.Println("\nMiniES stopped")
}
