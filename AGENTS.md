# Project: MiniES

## Tech Stack
- Language: Go 1.22+
- Module: github.com/rajal-ui/minies
- Build: `go build -o bin/minies ./cmd/minies`
- Run: `./bin/minies --listen=:8080 --shards=8 --ram-threshold=64MB --segment-dir=./data/segments --wal-path=./data/wal`

## Conventions
- Package structure follows internal/ layout
- All public functions have doc comments
- Use sync.RWMutex for concurrent index access
- Tokenization lowercases all tokens
- WAL is append-only; segments are immutable sorted files

## Commands
- Build: `go build -o bin/minies ./cmd/minies`
- Test: `go test ./...`
- Vet: `go vet ./...`
- Run: `./bin/minies`
- Lint: `gofmt -l .`
