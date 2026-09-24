// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package raptor

import (
	"context"
	"encoding/binary"
	"sync"

	"github.com/lemon4ksan/mach/proto/raptor"
)

// DatagramTransport defines the required QUIC methods to be wrapped.
type DatagramTransport interface {
	SendDatagram(p []byte) error
	ReceiveDatagram(ctx context.Context) ([]byte, error)
}

// Wrapper decorates a QUIC Connection to provide transparent RaptorQ fountain coding
// over DATAGRAM frames.
type Wrapper struct {
	conn    DatagramTransport
	encoder *raptor.Encoder
	decoder *raptor.Decoder

	sendMu  sync.Mutex
	sendSeq uint32
	block   [][]byte

	recvChan chan []byte
	errChan  chan error
}

// NewWrapper initializes a Wrapper over a datagram transport.
func NewWrapper(ctx context.Context, conn DatagramTransport) *Wrapper {
	w := &Wrapper{
		conn:     conn,
		encoder:  raptor.NewEncoder(),
		decoder:  raptor.NewDecoder(),
		block:    make([][]byte, 0, raptor.N),
		recvChan: make(chan []byte, 128),
		errChan:  make(chan error, 1),
	}

	w.decoder.ProcessAsync(ctx)
	go w.readLoop(ctx)

	return w
}

func (w *Wrapper) readLoop(ctx context.Context) {
	for {
		raw, err := w.conn.ReceiveDatagram(ctx)
		if err != nil {
			w.errChan <- err
			return
		}

		if len(raw) < 5 {
			continue
		}

		typ := raw[0]
		seq := binary.BigEndian.Uint32(raw[1:5])
		payload := raw[5:]

		switch typ {
		case 0x00:
			// Normal packet, pass it up immediately
			w.recvChan <- payload
			// Feed to decoder to help recovery of others
			w.decoder.Feed(raptor.Datagram{SymbolID: seq, Data: payload, IsRepair: false})
		case 0x01:
			// Repair packet
			w.decoder.Feed(raptor.Datagram{SymbolID: seq, Data: payload, IsRepair: true})
		}
	}
}

// SendDatagram intercepts data, sends it, and generates repair symbols periodically.
func (w *Wrapper) SendDatagram(p []byte) error {
	w.sendMu.Lock()
	defer w.sendMu.Unlock()

	// Send as data datagram: Type(1) | Seq(4) | Payload
	buf := make([]byte, 1+4+len(p))
	buf[0] = 0x00
	binary.BigEndian.PutUint32(buf[1:5], w.sendSeq)
	copy(buf[5:], p)

	if err := w.conn.SendDatagram(buf); err != nil {
		return err
	}

	w.block = append(w.block, p)
	w.sendSeq++

	if len(w.block) == raptor.N {
		// Generate repair symbol
		repair := w.encoder.Encode(w.block, w.sendSeq)
		w.block = w.block[:0]

		if len(repair) > 0 {
			rBuf := make([]byte, 1+4+len(repair))
			rBuf[0] = 0x01
			binary.BigEndian.PutUint32(rBuf[1:5], w.sendSeq)
			copy(rBuf[5:], repair)
			_ = w.conn.SendDatagram(rBuf) // Best effort send for repair symbols
		}
	}

	return nil
}

// ReceiveDatagram reads from the underlying transport or returns recovered packets.
func (w *Wrapper) ReceiveDatagram(ctx context.Context) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-w.errChan:
		return nil, err
	case p := <-w.recvChan:
		return p, nil
	case p := <-w.decoder.Recovered():
		return p, nil
	}
}
