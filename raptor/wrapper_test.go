// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package raptor

import (
	"context"
	"testing"
	"time"
)

// mockDatagramTransport is a simple mock for testing.
type mockDatagramTransport struct {
	sent     [][]byte
	recvChan chan []byte
}

func newMockTransport() *mockDatagramTransport {
	return &mockDatagramTransport{
		recvChan: make(chan []byte, 128),
	}
}

func (m *mockDatagramTransport) SendDatagram(p []byte) error {
	buf := make([]byte, len(p))
	copy(buf, p)
	m.sent = append(m.sent, buf)

	return nil
}

func (m *mockDatagramTransport) ReceiveDatagram(ctx context.Context) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case p := <-m.recvChan:
		return p, nil
	}
}

func TestWrapper_SendDatagram(t *testing.T) {
	transport := newMockTransport()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wrapper := NewWrapper(ctx, transport)

	// Send N packets to trigger repair symbol generation (N=32)
	for i := 0; i < 32; i++ {
		err := wrapper.SendDatagram([]byte{byte(i), byte(i + 1)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// We expect 32 data packets + 1 repair symbol = 33
	if len(transport.sent) != 33 {
		t.Fatalf("expected 33 datagrams sent, got %d", len(transport.sent))
	}

	// The last one should be a repair symbol (type 0x01)
	last := transport.sent[32]
	if len(last) < 5 || last[0] != 0x01 {
		t.Errorf("expected last datagram to be repair symbol (type 0x01)")
	}
}

func TestWrapper_ReceiveDatagram(t *testing.T) {
	transport := newMockTransport()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wrapper := NewWrapper(ctx, transport)

	// Send a mock regular packet (type 0x00, seq 1, payload "hello")
	mockPacket := []byte{0x00, 0x00, 0x00, 0x00, 0x01, 'h', 'e', 'l', 'l', 'o'}
	transport.recvChan <- mockPacket

	recvCtx, recvCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer recvCancel()

	data, err := wrapper.ReceiveDatagram(recvCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(data) != "hello" {
		t.Errorf("expected 'hello', got %s", string(data))
	}
}
