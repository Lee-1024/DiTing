package collector

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
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
	slog.Info("collector run-once starting", "path", c.path, "batch_size", c.batchSize)
	file, err := os.Open(c.path)
	if err != nil {
		slog.Error("collector open file failed", "path", c.path, "error", err)
		return err
	}
	defer file.Close()

	var batch []audit.Event
	var total int
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		event, err := ParseTetragonEvent(scanner.Bytes())
		if err != nil {
			if errors.Is(err, ErrUnsupportedEvent) {
				continue
			}
			slog.Error("collector parse event failed", "path", c.path, "error", err)
			return err
		}
		batch = append(batch, event)
		if len(batch) >= c.batchSize {
			slog.Info("collector writing batch", "path", c.path, "events", len(batch), "mode", "run_once")
			if err := c.writer.Write(ctx, batch); err != nil {
				slog.Error("collector write batch failed", "path", c.path, "events", len(batch), "error", err)
				return err
			}
			total += len(batch)
			batch = nil
		}
	}
	if err := scanner.Err(); err != nil {
		slog.Error("collector scan file failed", "path", c.path, "error", err)
		return err
	}
	if len(batch) > 0 {
		slog.Info("collector writing batch", "path", c.path, "events", len(batch), "mode", "run_once")
		if err := c.writer.Write(ctx, batch); err != nil {
			slog.Error("collector write batch failed", "path", c.path, "events", len(batch), "error", err)
			return err
		}
		total += len(batch)
	}
	slog.Info("collector run-once completed", "path", c.path, "events", total)
	return nil
}

func (c *FileCollector) Tail(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = time.Second
	}
	slog.Info("collector tail starting", "path", c.path, "batch_size", c.batchSize, "interval", interval.String(), "start_position", "end")

	file, info, err := c.openForTail(true)
	if err != nil {
		slog.Error("collector open file failed", "path", c.path, "error", err)
		return err
	}
	defer file.Close()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var partial []byte
	for {
		nextFile, nextInfo, err := c.reopenIfChanged(file, info)
		if err != nil {
			return err
		}
		if nextFile != file {
			slog.Info("collector reopened file", "path", c.path)
			_ = file.Close()
			file = nextFile
			partial = nil
		}
		info = nextInfo

		events, remaining, err := c.readAvailable(ctx, file, partial)
		if err != nil {
			return err
		}
		partial = remaining
		if len(events) > 0 {
			slog.Info("collector writing batch", "path", c.path, "events", len(events), "mode", "tail")
			if err := c.writer.Write(ctx, events); err != nil {
				slog.Error("collector write batch failed", "path", c.path, "events", len(events), "error", err)
				return err
			}
		}

		select {
		case <-ctx.Done():
			slog.Info("collector tail stopped", "path", c.path)
			return nil
		case <-ticker.C:
		}
	}
}

func (c *FileCollector) openForTail(seekEnd bool) (*os.File, os.FileInfo, error) {
	file, err := os.Open(c.path)
	if err != nil {
		return nil, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, err
	}
	if seekEnd {
		if _, err := file.Seek(0, io.SeekEnd); err != nil {
			_ = file.Close()
			return nil, nil, err
		}
	}
	return file, info, nil
}

func (c *FileCollector) reopenIfChanged(file *os.File, info os.FileInfo) (*os.File, os.FileInfo, error) {
	current, err := os.Stat(c.path)
	if err != nil {
		return file, info, err
	}
	offset, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return file, info, err
	}
	replacedOrTruncated := !os.SameFile(info, current) || current.Size() < offset
	rewrittenAtSameSize := os.SameFile(info, current) && current.Size() == offset && current.ModTime().After(info.ModTime())
	if !replacedOrTruncated && !rewrittenAtSameSize {
		return file, current, nil
	}
	return c.openForTail(false)
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
			slog.Error("collector parse event failed", "path", c.path, "error", err)
			return nil, nil, err
		}
		events = append(events, event)
		if len(events) >= c.batchSize {
			slog.Info("collector writing batch", "path", c.path, "events", len(events), "mode", "tail")
			if err := c.writer.Write(ctx, events); err != nil {
				slog.Error("collector write batch failed", "path", c.path, "events", len(events), "error", err)
				return nil, nil, err
			}
			events = nil
		}
	}

	return events, data[start:], nil
}
