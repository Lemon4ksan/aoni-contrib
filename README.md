
# 🧩 aoni-x: Ecosystem Extensions

**aoni-x** contains the official application-layer extensions and experimental protocols for the [aoni](https://github.com/lemon4ksan/aoni) Zero-Allocation HTTP engine.

Following the design philosophy of `golang.org/x/...`, this repository houses advanced enterprise standards, complex cryptography, and specialized network protocols that sit *above* the standard L3/L4/L7 networking stack. By keeping these out of the core `aoni` repository, we guarantee the core remains dependency-free and permanently frozen for strict backward compatibility.

## 📦 Packages

* **`auth/`**: Heavy application-layer cryptography and authentication standards.
  * **OAuth 2.0 DPoP** (RFC 9449): Demonstrating Proof-of-Possession at the Application Layer (JWS generation).
  * **OAuth 2.0 PKCE** (RFC 7636): Proof Key for Code Exchange (S256 verifiers).
  * **HTTP Message Signatures** (RFC 9421): Cryptographic request signing for Web3 and Banking APIs.
* **`geoip/`**: Fast MaxMind GeoIP2 / GeoLite2 MMDB geolocation database reader.
* **`otel/`**: Pure-Go OpenTelemetry (W3C TraceContext & OTLP/HTTP) distributed tracing.
* **`grpc/`**: High-performance gRPC client wrapper utilizing the core `aoni` transport.
* **`socketio/`**: Native Socket.IO v5 / Engine.IO v4 client implementation on top of `aoni` WebSockets.
* **`webtransport/`**: W3C and IETF WebTransport over HTTP/3 (RFC 9114, RFC 9297).
* **`webpush/`**: RFC 8030 / RFC 8188 Encrypted Content-Encoding (ECE) for WebPush notifications.
* **`sqlcookie/`**: SQL database-backed persistent cookie jar storage.
* **`ssh/`**: SSH tunneling over HTTP proxies.
* **`cmd/protoc-gen-aoni`**: Protobuf compiler plugin for generating `aoni`-compatible typed generic RPC clients.

## 🚀 Installation

Install only the specific extensions you need to keep your binary size minimal:

```bash
# Install the Socket.IO extension
go get github.com/lemon4ksan/aoni-x/socketio

# Install the DPoP and HTTP Signatures extension
go get github.com/lemon4ksan/aoni-x/auth
```

## 📖 Examples

### Socket.IO Client
```go
package main

import (
	"context"
	"fmt"
	
	"github.com/lemon4ksan/aoni"
	"github.com/lemon4ksan/aoni-x/socketio"
)

func main() {
	client := aoni.New()
	
	// Connect to a Socket.IO v5 server
	socket, _ := socketio.Dial(context.Background(), client, "wss://api.example.com/socket.io/")
	
	socket.On("message", func(data []byte) {
		fmt.Printf("Received: %s\n", data)
	})
	
	socket.Emit("join", []byte(`{"room": "chat"}`))
	
	select {}
}
```

### OAuth 2.0 DPoP (RFC 9449)
```go
package main

import (
	"context"
	
	"github.com/lemon4ksan/aoni"
	"github.com/lemon4ksan/aoni-x/auth/mod"
)

func main() {
	client := aoni.New()
	
	// Automatically generates and signs a DPoP Proof JWT for this specific request
	client.Get(context.Background(), "https://api.example.com/protected", 
		mod.WithDPoPToken("access_token_123", myPrivateKey),
	)
}
```

