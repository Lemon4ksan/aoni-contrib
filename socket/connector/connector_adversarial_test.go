// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package connector_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"

	"github.com/lemon4ksan/aoni-contrib/socket"
	"github.com/lemon4ksan/aoni-contrib/socket/connector"
)

// TestConnector_ReconnectPolicy_NilEndpointSelector_FallsBackToLastEndpoint verifies that
// a custom ReconnectPolicy without EndpointSelector does not panic in reconnectLoop and
// cleanly falls back to c.lastEndpoint.
func TestConnector_ReconnectPolicy_NilEndpointSelector_FallsBackToLastEndpoint(t *testing.T) {
	t.Parallel()

	var (
		dialMu        sync.Mutex
		dialEndpoints []string
	)

	dialer := func(_ context.Context, endpoint string, _ socket.Framer, _ socket.Cipher) (connector.Connection, error) {
		dialMu.Lock()

		dialEndpoints = append(dialEndpoints, endpoint)
		dialMu.Unlock()

		return &mockConnection{
			incoming: make(chan *socket.FrameBuffer, 10),
		}, nil
	}

	cfg := connector.Config[string]{
		Dialer: dialer,
		ReconnectPolicy: connector.ReconnectPolicy[string]{
			InitialBackoff: 10 * time.Millisecond,
			MaxBackoff:     50 * time.Millisecond,
			// EndpointSelector is intentionally nil
		},
	}

	c := connector.New[string](cfg)
	defer func() { _ = c.Close() }()

	err := c.Connect(t.Context(), "127.0.0.1:8080")
	require.NoError(t, err)

	// Simulate connection drop
	err = c.Disconnect()
	require.NoError(t, err)
	assert.False(t, c.IsConnected())

	// Trigger reconnect with nil EndpointSelector: must NOT panic, must fall back to lastEndpoint
	c.TriggerReconnect()

	require.Eventually(t, func() bool {
		return c.IsConnected()
	}, 2*time.Second, 10*time.Millisecond, "connector should reconnect via reconnectLoop")

	dialMu.Lock()
	defer dialMu.Unlock()

	require.GreaterOrEqual(t, len(dialEndpoints), 2)
	assert.Equal(t, "127.0.0.1:8080", dialEndpoints[1], "reconnect attempt must fall back to lastEndpoint")
}

// TestConnector_ReconnectPolicy_NilEndpointSelector_MultiCycleStress stress-tests
// 25 sequential cycles of disconnect and TriggerReconnect under a policy with nil EndpointSelector.
func TestConnector_ReconnectPolicy_NilEndpointSelector_MultiCycleStress(t *testing.T) {
	t.Parallel()

	var (
		dialCount atomic.Int32
		lastDial  atomic.Value
	)

	dialer := func(_ context.Context, endpoint string, _ socket.Framer, _ socket.Cipher) (connector.Connection, error) {
		dialCount.Add(1)
		lastDial.Store(endpoint)

		return &mockConnection{
			incoming: make(chan *socket.FrameBuffer, 10),
		}, nil
	}

	cfg := connector.Config[string]{
		Dialer: dialer,
		ReconnectPolicy: connector.ReconnectPolicy[string]{
			InitialBackoff: 5 * time.Millisecond,
			MaxBackoff:     20 * time.Millisecond,
		},
	}

	c := connector.New[string](cfg)
	defer func() { _ = c.Close() }()

	initialTarget := "192.168.1.100:9000"
	err := c.Connect(t.Context(), initialTarget)
	require.NoError(t, err)

	for cycle := 0; cycle < 25; cycle++ {
		t.Logf("Starting cycle %d", cycle)

		err = c.Disconnect()
		require.NoError(t, err)

		c.TriggerReconnect()

		require.Eventually(t, func() bool {
			return c.IsConnected()
		}, 2*time.Second, 5*time.Millisecond, fmt.Sprintf("failed on cycle %d", cycle))

		assert.Equal(t, initialTarget, lastDial.Load().(string))
		t.Logf("Completed cycle %d", cycle)
	}

	assert.GreaterOrEqual(t, dialCount.Load(), int32(26))
}

// TestConnector_WaitForConnection_ConcurrentStress tests multiple goroutines
// waiting on connection completion simultaneously.
func TestConnector_WaitForConnection_ConcurrentStress(t *testing.T) {
	t.Parallel()

	mockConn := &mockConnection{
		incoming: make(chan *socket.FrameBuffer, 10),
	}

	dialBlock := make(chan struct{})

	var dialCount atomic.Int32

	dialer := func(_ context.Context, endpoint string, _ socket.Framer, _ socket.Cipher) (connector.Connection, error) {
		dialCount.Add(1)
		<-dialBlock
		return mockConn, nil
	}

	cfg := connector.Config[string]{
		Dialer: dialer,
	}

	c := connector.New[string](cfg)
	defer func() { _ = c.Close() }()

	// Start dial in background
	go func() {
		_ = c.Connect(context.Background(), "127.0.0.1:8080")
	}()

	// Wait until IsConnecting is true
	require.Eventually(t, func() bool {
		return c.IsConnecting()
	}, 1*time.Second, 5*time.Millisecond)

	// Launch 20 concurrent waiters
	var wg sync.WaitGroup

	errs := make([]error, 20)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			errs[idx] = c.WaitForConnection(ctx)
		}(i)
	}

	// Release dialer
	time.Sleep(20 * time.Millisecond)
	close(dialBlock)

	wg.Wait()

	for i := 0; i < 20; i++ {
		assert.NoError(t, errs[i], fmt.Sprintf("waiter %d should succeed", i))
	}

	assert.True(t, c.IsConnected())
	ep, ok := c.CurrentEndpoint()
	assert.True(t, ok)
	assert.Equal(t, "127.0.0.1:8080", ep)
}

// TestConnector_WaitForConnection_DialFailure tests that waiters receive ErrDisconnected
// when an in-progress dial fails.
func TestConnector_WaitForConnection_DialFailure(t *testing.T) {
	t.Parallel()

	dialBlock := make(chan struct{})
	dialer := func(_ context.Context, endpoint string, _ socket.Framer, _ socket.Cipher) (connector.Connection, error) {
		<-dialBlock
		return nil, errors.New("dial failed")
	}

	cfg := connector.Config[string]{
		Dialer: dialer,
	}

	c := connector.New[string](cfg)
	defer func() { _ = c.Close() }()

	go func() {
		_ = c.Connect(context.Background(), "127.0.0.1:8080")
	}()

	require.Eventually(t, func() bool {
		return c.IsConnecting()
	}, 1*time.Second, 5*time.Millisecond)

	var wg sync.WaitGroup

	errs := make([]error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			errs[idx] = c.WaitForConnection(ctx)
		}(i)
	}

	close(dialBlock)
	wg.Wait()

	for i := 0; i < 10; i++ {
		assert.ErrorIs(t, errs[i], connector.ErrDisconnected, fmt.Sprintf("waiter %d should report ErrDisconnected", i))
	}
}

// TestConnector_CurrentEndpoint_ThreadSafety tests concurrent access to CurrentEndpoint,
// IsConnected, and IsConnecting while connections and disconnections cycle.
func TestConnector_CurrentEndpoint_ThreadSafety(t *testing.T) {
	t.Parallel()

	mockConn := &mockConnection{
		incoming: make(chan *socket.FrameBuffer, 10),
	}

	dialer := func(_ context.Context, endpoint string, _ socket.Framer, _ socket.Cipher) (connector.Connection, error) {
		return mockConn, nil
	}

	cfg := connector.Config[string]{
		Dialer: dialer,
	}

	c := connector.New[string](cfg)
	defer func() { _ = c.Close() }()

	stop := make(chan struct{})

	var wg sync.WaitGroup

	// Reader goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-stop:
					return
				default:
					_, _ = c.CurrentEndpoint()
					_ = c.IsConnected()
					_ = c.IsConnecting()
				}
			}
		}()
	}

	// Cycling connect/disconnect
	for i := 0; i < 10; i++ {
		_ = c.Connect(t.Context(), "10.0.0.1:80")
		_ = c.Disconnect()
	}

	close(stop)
	wg.Wait()
}
