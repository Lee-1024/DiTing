package collector

import (
	"context"
	"fmt"
	"strings"

	tetragon "github.com/cilium/tetragon/api/v1/tetragon"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type TetragonObserverManager struct {
	addr string
}

func NewTetragonObserverManager(addr string) (*TetragonObserverManager, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil, fmt.Errorf("Tetragon gRPC address is empty")
	}
	return &TetragonObserverManager{addr: addr}, nil
}

func (m *TetragonObserverManager) Apply(ctx context.Context, policy string) error {
	client, closeConn, err := m.client(ctx)
	if err != nil {
		return err
	}
	defer closeConn()
	if _, err := client.DeleteTracingPolicy(ctx, &tetragon.DeleteTracingPolicyRequest{Name: tetragonObserverPolicyName}); err != nil && status.Code(err) != codes.NotFound {
		return fmt.Errorf("delete previous Tetragon observer policy: %w", err)
	}
	if _, err := client.AddTracingPolicy(ctx, &tetragon.AddTracingPolicyRequest{Yaml: policy}); err != nil {
		return fmt.Errorf("add Tetragon observer policy: %w", err)
	}
	return nil
}

func (m *TetragonObserverManager) Remove(ctx context.Context) error {
	client, closeConn, err := m.client(ctx)
	if err != nil {
		return err
	}
	defer closeConn()
	_, err = client.DeleteTracingPolicy(ctx, &tetragon.DeleteTracingPolicyRequest{Name: tetragonObserverPolicyName})
	if status.Code(err) == codes.NotFound {
		return nil
	}
	return err
}

func (m *TetragonObserverManager) client(ctx context.Context) (tetragon.FineGuidanceSensorsClient, func() error, error) {
	conn, err := grpc.NewClient(m.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("connect Tetragon gRPC: %w", err)
	}
	return tetragon.NewFineGuidanceSensorsClient(conn), conn.Close, nil
}
