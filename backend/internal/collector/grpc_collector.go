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
	flushInterval     time.Duration
	reconnectInterval time.Duration
	dial              func(context.Context, string) (eventStream, func() error, error)
	afterWrite        func()
	onConnect         func()
	onError           func(error)
	afterErrorForTest func()
}

// NewGRPCCollector 创建并初始化 New GRPCCollector 实例。
func NewGRPCCollector(addr string, batchSize int, writer EventWriter) *GRPCCollector {
	if batchSize <= 0 {
		batchSize = 1000
	}
	collector := &GRPCCollector{
		addr:              addr,
		batchSize:         batchSize,
		flushInterval:     time.Second,
		writer:            writer,
		reconnectInterval: 5 * time.Second,
	}
	collector.dial = collector.dialTetragon
	return collector
}

// SetReconnectInterval 设置 Set Reconnect Interval。
func (c *GRPCCollector) SetReconnectInterval(interval time.Duration) {
	if interval > 0 {
		c.reconnectInterval = interval
	}
}

// SetFlushInterval 设置 Set Flush Interval。
func (c *GRPCCollector) SetFlushInterval(interval time.Duration) {
	if interval > 0 {
		c.flushInterval = interval
	}
}

// SetErrorHandler 设置 Set Error Handler。
func (c *GRPCCollector) SetErrorHandler(handler func(error)) {
	c.onError = handler
}

// SetConnectHandler 设置 Set Connect Handler。
func (c *GRPCCollector) SetConnectHandler(handler func()) {
	c.onConnect = handler
}

// RunOnce 运行 Run Once 的主流程。
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

// Run 运行 Run 的主流程。
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
		slog.Info("collector grpc stream opened", "addr", c.addr)
		c.reportConnect()
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

// consume 处理 consume 相关逻辑。
func (c *GRPCCollector) consume(ctx context.Context, stream eventStream) error {
	var batch []audit.Event
	var received uint64
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		slog.Info("collector grpc writing batch", "addr", c.addr, "events", len(batch))
		if err := c.writer.Write(ctx, batch); err != nil {
			return err
		}
		if c.afterWrite != nil {
			c.afterWrite()
		}
		batch = nil
		return nil
	}

	responses := make(chan *tetragon.GetEventsResponse)
	errs := make(chan error, 1)
	go func() {
		defer close(responses)
		for {
			response, err := stream.Recv()
			if err != nil {
				errs <- err
				return
			}
			select {
			case responses <- response:
			case <-ctx.Done():
				errs <- ctx.Err()
				return
			}
		}
	}()

	ticker := time.NewTicker(c.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if flushErr := flush(); flushErr != nil {
				c.reportError(flushErr)
				return flushErr
			}
			return ctx.Err()
		case <-ticker.C:
			if flushErr := flush(); flushErr != nil {
				c.reportError(flushErr)
				return flushErr
			}
			continue
		case err := <-errs:
			if errors.Is(err, io.EOF) {
				if flushErr := flush(); flushErr != nil {
					c.reportError(flushErr)
					return flushErr
				}
				return io.EOF
			}
			if flushErr := flush(); flushErr != nil {
				c.reportError(flushErr)
				return flushErr
			}
			return err
		case response, ok := <-responses:
			if !ok {
				continue
			}
			event, err := ParseTetragonGRPCEvent(response)
			if err != nil {
				if errors.Is(err, ErrUnsupportedEvent) {
					slog.Debug("collector grpc skipped unsupported event", "addr", c.addr)
					continue
				}
				c.reportError(err)
				return err
			}
			received++
			if received == 1 {
				slog.Info("collector grpc received first supported event", "addr", c.addr, "event_type", event.EventType, "node_name", event.NodeName)
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
}

// reportError 处理 report Error 相关逻辑。
func (c *GRPCCollector) reportError(err error) {
	if err != nil && c.onError != nil {
		c.onError(err)
	}
	if err != nil && c.afterErrorForTest != nil {
		c.afterErrorForTest()
	}
}

// reportConnect 处理 report Connect 相关逻辑。
func (c *GRPCCollector) reportConnect() {
	if c.onConnect != nil {
		c.onConnect()
	}
}

// dialTetragon 处理 dial Tetragon 相关逻辑。
func (c *GRPCCollector) dialTetragon(ctx context.Context, addr string) (eventStream, func() error, error) {
	slog.Info("collector grpc connecting", "addr", addr)
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

// waitContext 处理 wait Context 相关逻辑。
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
