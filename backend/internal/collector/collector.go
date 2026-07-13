package collector

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"time"

	"diting/backend/internal/audit"
)

type EventWriter interface {
	Write(ctx context.Context, events []audit.Event) error
}

type FileCollector struct {
	path      string
	batchSize int
	writer    EventWriter
}

func NewFileCollector(path string, batchSize int, writer EventWriter) *FileCollector {
	if batchSize <= 0 {
		batchSize = 1000
	}
	return &FileCollector{path: path, batchSize: batchSize, writer: writer}
}

func (c *FileCollector) RunOnce(ctx context.Context) error {
	file, err := os.Open(c.path)
	if err != nil {
		return err
	}
	defer file.Close()

	var batch []audit.Event
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		event, err := ParseTetragonEvent(scanner.Bytes())
		if err != nil {
			if errors.Is(err, ErrUnsupportedEvent) {
				continue
			}
			return err
		}
		batch = append(batch, event)
		if len(batch) >= c.batchSize {
			if err := c.writer.Write(ctx, batch); err != nil {
				return err
			}
			batch = nil
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if len(batch) > 0 {
		return c.writer.Write(ctx, batch)
	}
	return nil
}

func (c *FileCollector) Tail(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = time.Second
	}

	file, err := os.Open(c.path)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return err
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var partial []byte
	for {
		events, remaining, err := c.readAvailable(ctx, file, partial)
		if err != nil {
			return err
		}
		partial = remaining
		if len(events) > 0 {
			if err := c.writer.Write(ctx, events); err != nil {
				return err
			}
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (c *FileCollector) readAvailable(ctx context.Context, file *os.File, partial []byte) ([]audit.Event, []byte, error) {
	buffer := make([]byte, 32*1024)
	var events []audit.Event
	data := append([]byte{}, partial...)

	for {
		n, err := file.Read(buffer)
		if n > 0 {
			data = append(data, buffer[:n]...)
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, err
		}
	}

	start := 0
	for i, b := range data {
		if b != '\n' {
			continue
		}
		line := data[start:i]
		start = i + 1
		if len(line) == 0 {
			continue
		}
		event, err := ParseTetragonEvent(line)
		if err != nil {
			if errors.Is(err, ErrUnsupportedEvent) {
				continue
			}
			return nil, nil, err
		}
		events = append(events, event)
		if len(events) >= c.batchSize {
			if err := c.writer.Write(ctx, events); err != nil {
				return nil, nil, err
			}
			events = nil
		}
	}

	return events, data[start:], nil
}
