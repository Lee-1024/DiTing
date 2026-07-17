package collector

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"time"

	"diting/backend/internal/audit"
	tetragon "github.com/cilium/tetragon/api/v1/tetragon"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type eventStream interface {
	Recv() (*tetragon.GetEventsResponse, error)
}

type GRPCCollector struct {
	addr              string
	batchSize         int
	writer            EventWriter
	reconnectInterval time.Duration
	dial              func(context.Context, string) (eventStream, func() error, error)
	afterWrite        func()
	onError           func(error)
	afterErrorForTest func()
}

func NewGRPCCollector(addr string, batchSize int, writer EventWriter) *GRPCCollector {
	if batchSize <= 0 {
		batchSize = 1000
	}
	collector := &GRPCCollector{
		addr:              addr,
		batchSize:         batchSize,
		writer:            writer,
		reconnectInterval: 5 * time.Second,
	}
	collector.dial = collector.dialTetragon
	return collector
}

func (c *GRPCCollector) SetReconnectInterval(interval time.Duration) {
	if interval > 0 {
		c.reconnectInterval = interval
	}
}

func (c *GRPCCollector) SetErrorHandler(handler func(error)) {
	c.onError = handler
}

func (c *GRPCCollector) RunOnce(ctx context.Context) error {
	stream, closeConn, err := c.dial(ctx, c.addr)
	if err != nil {
		return err
	}
	defer func() { _ = closeConn() }()
	if err := c.consume(ctx, stream); errors.Is(err, io.EOF) {
		return nil
	} else {
		return err
	}
}

func (c *GRPCCollector) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		stream, closeConn, err := c.dial(ctx, c.addr)
		if err != nil {
			slog.Error("collector grpc connect failed", "addr", c.addr, "error", err)
			c.reportError(err)
			if waitErr := waitContext(ctx, c.reconnectInterval); waitErr != nil {
				return nil
			}
			continue
		}
		err = c.consume(ctx, stream)
		_ = closeConn()
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil
		}
		if errors.Is(err, io.EOF) {
			slog.Warn("collector grpc stream closed", "addr", c.addr)
		} else {
			slog.Error("collector grpc stream failed", "addr", c.addr, "error", err)
			c.reportError(err)
		}
		if waitErr := waitContext(ctx, c.reconnectInterval); waitErr != nil {
			return nil
		}
	}
}

func (c *GRPCCollector) consume(ctx context.Context, stream eventStream) error {
	var batch []audit.Event
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := c.writer.Write(ctx, batch); err != nil {
			return err
		}
		if c.afterWrite != nil {
			c.afterWrite()
		}
		batch = nil
		return nil
	}

	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			if flushErr := flush(); flushErr != nil {
				c.reportError(flushErr)
				return flushErr
			}
			return io.EOF
		}
		if err != nil {
			if flushErr := flush(); flushErr != nil {
				c.reportError(flushErr)
				return flushErr
			}
			return err
		}
		event, err := ParseTetragonGRPCEvent(response)
		if err != nil {
			if errors.Is(err, ErrUnsupportedEvent) {
				continue
			}
			c.reportError(err)
			return err
		}
		batch = append(batch, event)
		if len(batch) >= c.batchSize {
			if err := flush(); err != nil {
				c.reportError(err)
				return err
			}
		}
	}
}

func (c *GRPCCollector) reportError(err error) {
	if err != nil && c.onError != nil {
		c.onError(err)
	}
	if err != nil && c.afterErrorForTest != nil {
		c.afterErrorForTest()
	}
}

func (c *GRPCCollector) dialTetragon(ctx context.Context, addr string) (eventStream, func() error, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	client := tetragon.NewFineGuidanceSensorsClient(conn)
	stream, err := client.GetEvents(ctx, &tetragon.GetEventsRequest{})
	if err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	return stream, conn.Close, nil
}

func waitContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
