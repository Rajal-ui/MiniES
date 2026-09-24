package storage

import (
	"bufio"
	"encoding/binary"
	"os"
	"sync"
)

const (
	walMagic = 0x4D494E45
	version  = 1
)

type WAL struct {
	path   string
	file   *os.File
	writer *bufio.Writer
	mu     sync.Mutex
}

func NewWAL(path string) (*WAL, error) {
	os.MkdirAll(path, 0755)
	walPath := path + "/wal.log"
	f, err := os.OpenFile(walPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	w := &WAL{
		path:   walPath,
		file:   f,
		writer: bufio.NewWriter(f),
	}
	return w, nil
}

func (w *WAL) Append(record []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	header := make([]byte, 16)
	binary.BigEndian.PutUint32(header[0:4], walMagic)
	binary.BigEndian.PutUint32(header[4:8], version)
	binary.BigEndian.PutUint32(header[8:12], uint32(len(record)))
	binary.BigEndian.PutUint32(header[12:16], crc32(record))
	if _, err := w.writer.Write(header); err != nil {
		return err
	}
	if _, err := w.writer.Write(record); err != nil {
		return err
	}
	return nil
}

func (w *WAL) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writer.Flush()
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.writer != nil {
		w.writer.Flush()
	}
	return w.file.Close()
}

func (w *WAL) Replay() ([][]byte, error) {
	f, err := os.Open(w.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var records [][]byte
	reader := bufio.NewReader(f)
	for {
		header := make([]byte, 16)
		_, err := reader.Read(header)
		if err != nil {
			break
		}
		magic := binary.BigEndian.Uint32(header[0:4])
		if magic != walMagic {
			continue
		}
		length := binary.BigEndian.Uint32(header[8:12])
		record := make([]byte, length)
		_, err = reader.Read(record)
		if err != nil {
			break
		}
		records = append(records, record)
	}
	return records, nil
}

func crc32(data []byte) uint32 {
	var crc uint32 = 0xFFFFFFFF
	for _, b := range data {
		crc ^= uint32(b)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0xEDB88320
			} else {
				crc >>= 1
			}
		}
	}
	return ^crc
}
