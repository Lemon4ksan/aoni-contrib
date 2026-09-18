// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grpc

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/lemon4ksan/foundation/net/http/header"
	"google.golang.org/protobuf/proto"

	"github.com/lemon4ksan/aoni"
	"github.com/lemon4ksan/aoni/x/codec/decode"
	"github.com/lemon4ksan/aoni/mod"
)

// Re-exported Protobuf and gRPC-Web decoders for single-package discoverability.
var (
	// ProtoDecoder reads binary Protocol Buffer payloads into proto.Message targets.
	ProtoDecoder = decode.ProtoDecoder

	// GRPCWebDecoder extracts Protobuf payloads from 5-byte gRPC-Web frames and validates trailers.
	GRPCWebDecoder = decode.GRPCWebDecoder

	// ProtoJSONDecoder parses JSON response streams into Protobuf messages via protojson.
	ProtoJSONDecoder = decode.ProtoJSONDecoder

	// WithProto creates a [aoni.RequestModifier] that assigns ProtoDecoder for response parsing.
	WithProto = decode.WithProto

	// WithGRPCWeb creates a [aoni.RequestModifier] that assigns GRPCWebDecoder for response parsing.
	WithGRPCWeb = decode.WithGRPCWeb
)

// WithGRPCWebTimeout assigns standard gRPC-Web timeout headers.
func WithGRPCWebTimeout(d time.Duration) aoni.RequestModifier {
	return mod.WithHeader(header.GRPCTimeout, formatGRPCTimeout(d))
}

// ProtoGetTo executes an HTTP GET request expecting a binary Protocol Buffer response stream unmarshaled into Resp.
func ProtoGetTo[Resp any](
	ctx context.Context,
	doer aoni.HTTPRequester,
	path string,
	mods ...aoni.RequestModifier,
) (*Resp, error) {
	var stackBuf [stackModCapacity]aoni.RequestModifier

	allMods := withProtoGetMods(&stackBuf, mods)

	resp, err := doer.Request(ctx, http.MethodGet, path, allMods...)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result := new(Resp)
	if err := decode.ProtoDecoder.Decode(resp.Body, result); err != nil {
		return nil, err
	}

	return result, nil
}

// ProtoGetInto executes an HTTP GET request expecting a binary Protocol Buffer stream unmarshaled directly into target.
func ProtoGetInto[T any](
	ctx context.Context,
	doer aoni.HTTPRequester,
	path string,
	target *T,
	mods ...aoni.RequestModifier,
) error {
	var stackBuf [stackModCapacity]aoni.RequestModifier

	allMods := withProtoGetMods(&stackBuf, mods)

	resp, err := doer.Request(ctx, http.MethodGet, path, allMods...)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return decode.ProtoDecoder.Decode(resp.Body, target)
}

// ProtoPostTo executes an HTTP POST request carrying a binary [proto.Message] payload and unmarshals the response into Resp.
func ProtoPostTo[Resp any](
	ctx context.Context,
	doer aoni.HTTPRequester,
	path string,
	msg proto.Message,
	mods ...aoni.RequestModifier,
) (*Resp, error) {
	var stackBuf [stackModCapacity]aoni.RequestModifier

	allMods := withProtoPostMods(&stackBuf, msg, mods)

	resp, err := doer.Request(ctx, http.MethodPost, path, allMods...)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result := new(Resp)
	if err := decode.ProtoDecoder.Decode(resp.Body, result); err != nil {
		return nil, err
	}

	return result, nil
}

// ProtoPostInto executes an HTTP POST request carrying a binary [proto.Message] payload and unmarshals the response directly into target.
func ProtoPostInto[T any](
	ctx context.Context,
	doer aoni.HTTPRequester,
	path string,
	msg proto.Message,
	target *T,
	mods ...aoni.RequestModifier,
) error {
	var stackBuf [stackModCapacity]aoni.RequestModifier

	allMods := withProtoPostMods(&stackBuf, msg, mods)

	resp, err := doer.Request(ctx, http.MethodPost, path, allMods...)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return decode.ProtoDecoder.Decode(resp.Body, target)
}

// WebPostTo executes an HTTP POST request carrying a 5-byte gRPC-Web framed payload and unmarshals the response into Resp.
func WebPostTo[Resp any](
	ctx context.Context,
	doer aoni.HTTPRequester,
	path string,
	msg proto.Message,
	mods ...aoni.RequestModifier,
) (*Resp, error) {
	var stackBuf [stackModCapacity]aoni.RequestModifier

	allMods := withWebPostMods(&stackBuf, msg, mods)

	resp, err := doer.Request(ctx, http.MethodPost, path, allMods...)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result := new(Resp)
	if err := decode.GRPCWebDecoder.Decode(resp.Body, result); err != nil {
		return nil, err
	}

	return result, nil
}

// WebPostInto executes an HTTP POST request carrying a 5-byte gRPC-Web framed payload and unmarshals the response directly into target.
func WebPostInto[T any](
	ctx context.Context,
	doer aoni.HTTPRequester,
	path string,
	msg proto.Message,
	target *T,
	mods ...aoni.RequestModifier,
) error {
	var stackBuf [stackModCapacity]aoni.RequestModifier

	allMods := withWebPostMods(&stackBuf, msg, mods)

	resp, err := doer.Request(ctx, http.MethodPost, path, allMods...)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return decode.GRPCWebDecoder.Decode(resp.Body, target)
}

const stackModCapacity = 16

func withProtoGetMods(
	stackBuf *[stackModCapacity]aoni.RequestModifier,
	mods []aoni.RequestModifier,
) []aoni.RequestModifier {
	total := 2 + len(mods)
	if total <= stackModCapacity {
		stackBuf[0] = mod.WithHeader(header.Accept, header.MIMEApplicationProtobuf)
		stackBuf[1] = mod.WithDecoder(decode.ProtoDecoder)
		copy(stackBuf[2:], mods)

		return stackBuf[:total]
	}

	allMods := make([]aoni.RequestModifier, 0, total)
	allMods = append(
		allMods,
		mod.WithHeader(header.Accept, header.MIMEApplicationProtobuf),
		mod.WithDecoder(decode.ProtoDecoder),
	)
	allMods = append(allMods, mods...)

	return allMods
}

func withProtoPostMods(
	stackBuf *[stackModCapacity]aoni.RequestModifier,
	msg proto.Message,
	mods []aoni.RequestModifier,
) []aoni.RequestModifier {
	bodyBytes, _ := proto.Marshal(msg)

	total := 3 + len(mods)
	if total <= stackModCapacity {
		stackBuf[0] = mod.WithBodyBytes(bodyBytes)
		stackBuf[1] = mod.WithHeader(header.ContentType, header.MIMEApplicationProtobuf)
		stackBuf[2] = mod.WithDecoder(decode.ProtoDecoder)
		copy(stackBuf[3:], mods)

		return stackBuf[:total]
	}

	allMods := make([]aoni.RequestModifier, 0, total)
	allMods = append(
		allMods,
		mod.WithBodyBytes(bodyBytes),
		mod.WithHeader(header.ContentType, header.MIMEApplicationProtobuf),
		mod.WithDecoder(decode.ProtoDecoder),
	)
	allMods = append(allMods, mods...)

	return allMods
}

func withWebPostMods(
	stackBuf *[stackModCapacity]aoni.RequestModifier,
	msg proto.Message,
	mods []aoni.RequestModifier,
) []aoni.RequestModifier {
	frameBytes, _ := MarshalFrame(msg, false)

	total := 3 + len(mods)
	if total <= stackModCapacity {
		stackBuf[0] = mod.WithBodyBytes(frameBytes)
		stackBuf[1] = mod.WithHeader(header.ContentType, header.MIMEApplicationGRPCWebProto)
		stackBuf[2] = mod.WithDecoder(decode.GRPCWebDecoder)
		copy(stackBuf[3:], mods)

		return stackBuf[:total]
	}

	allMods := make([]aoni.RequestModifier, 0, total)
	allMods = append(
		allMods,
		mod.WithBodyBytes(frameBytes),
		mod.WithHeader(header.ContentType, header.MIMEApplicationGRPCWebProto),
		mod.WithDecoder(decode.GRPCWebDecoder),
	)
	allMods = append(allMods, mods...)

	return allMods
}

// formatGRPCTimeout converts d into a PROTOCOL-HTTP2.md compliant "grpc-timeout" header string.
func formatGRPCTimeout(d time.Duration) string {
	if d <= 0 {
		return "0m"
	}

	switch {
	case d < time.Microsecond:
		return strconv.FormatInt(d.Nanoseconds(), 10) + "n"
	case d < time.Millisecond:
		return strconv.FormatInt(d.Microseconds(), 10) + "u"
	case d < time.Second:
		return strconv.FormatInt(d.Milliseconds(), 10) + "m"
	case d < time.Minute:
		return strconv.FormatInt(int64(d.Seconds()), 10) + "S"
	case d < time.Hour:
		return strconv.FormatInt(int64(d.Minutes()), 10) + "M"
	default:
		return strconv.FormatInt(int64(d.Hours()), 10) + "H"
	}
}
