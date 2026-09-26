package storage

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type Segment struct {
	ID       int
	Path     string
	Count    int
	Checksum uint32
}

func WriteSegment(segmentID int, dir string, records [][]byte) (*Segment, error) {
	os.MkdirAll(dir, 0755)
	path := fmt.Sprintf("%s/segment_%06d.bin", dir, segmentID)
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	writer := bufio.NewWriter(f)
	for _, rec := range records {
		if err := binary.Write(writer, binary.LittleEndian, uint32(len(rec))); err != nil {
			return nil, err
		}
		if _, err := writer.Write(rec); err != nil {
			return nil, err
		}
	}
	writer.Flush()

	return &Segment{
		ID:    segmentID,
		Path:  path,
		Count: len(records),
	}, nil
}

func ReadSegment(path string) ([][]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var records [][]byte
	reader := bufio.NewReader(f)
	for {
		var length uint32
		if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
			break
		}
		record := make([]byte, length)
		if _, err := io.ReadFull(reader, record); err != nil {
			break
		}
		records = append(records, record)
	}
	return records, nil
}
