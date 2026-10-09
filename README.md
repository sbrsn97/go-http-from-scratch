# go-http-from-scratch

[![CI](https://github.com/sbrsn97/go-http-from-scratch/actions/workflows/ci.yml/badge.svg)](https://github.com/sbrsn97/go-http-from-scratch/actions/workflows/ci.yml)

An HTTP/1.1 server implemented from scratch in Go on top of raw TCP connections.

The project intentionally does **not** use Go's `net/http` server implementation. Its purpose is to explore how HTTP actually works below the framework level: TCP streams, request framing, parsing, persistent connections, concurrency, timeouts, body encoding, and response serialization.

This is a learning and portfolio project rather than a production HTTP server.

## Why I Built This

Backend frameworks usually hide most of the networking and protocol details involved in serving an HTTP request.

I wanted to understand what happens underneath abstractions such as ASP.NET Core or Go's `net/http`:

```text
TCP connection
    ↓
byte stream
    ↓
HTTP message framing
    ↓
request parsing
    ↓
routing
    ↓
response serialization
    ↓
TCP connection
```

Building the server incrementally exposed several details that are easy to miss when working only at framework level, especially that TCP is a byte stream rather than a message-oriented protocol.

A single HTTP request can arrive across multiple `Read` calls, while a single `Read` may also contain bytes belonging to multiple HTTP requests.

The server therefore performs its own application-level framing instead of assuming that one TCP read corresponds to one HTTP message.

## Features

- Raw TCP listener using Go's `net` package
- HTTP/1.1 request-line parsing
- Case-insensitive header handling
- Request-target parsing into path and raw query
- `Host` validation for HTTP/1.1 requests
- `Content-Length` request body framing
- Chunked request body decoding
- Chunked response encoding
- Persistent HTTP/1.1 connections
- Multiple requests buffered on a single TCP connection
- Concurrent clients using goroutines
- Read and write deadlines
- Request header and body size limits
- Malformed request handling with HTTP error responses
- Method + path routing
- Static file serving with path traversal protection
- Automatic response `Content-Length`
- Configurable request logging middleware
- Graceful shutdown
- Unit tests
- Connection-level integration tests using `net.Pipe`
- Local load testing with Bombardier

## Example Endpoints

```text
GET  /hello          Returns a simple text response
POST /echo           Echoes the decoded request body
GET  /chunked        Returns a chunked HTTP response
GET  /               Serves public/index.html
GET  /static/...     Serves files under public/static
```

For example:

```powershell
curl.exe -i http://localhost:8080/hello
```

```text
HTTP/1.1 200 OK
Content-Type: text/plain
Content-Length: ...

hello from our HTTP server
```

The echo endpoint supports both `Content-Length` and chunked request bodies.

```powershell
curl.exe -i -X POST http://localhost:8080/echo -d "hello"
```

## Architecture

```text
                    ┌─────────────────────┐
                    │      TCP Client     │
                    └──────────┬──────────┘
                               │
                               │ byte stream
                               ▼
                    ┌─────────────────────┐
                    │   net.Listener      │
                    └──────────┬──────────┘
                               │
                               │ Accept()
                               ▼
                    ┌─────────────────────┐
                    │ connection goroutine│
                    └──────────┬──────────┘
                               │
                               ▼
                    ┌─────────────────────┐
                    │   requestReader     │
                    │                     │
                    │ framing + buffering │
                    └──────────┬──────────┘
                               │
                               ▼
                    ┌─────────────────────┐
                    │      Request        │
                    │                     │
                    │ method              │
                    │ path / query        │
                    │ headers             │
                    │ body                │
                    └──────────┬──────────┘
                               │
                               ▼
                    ┌─────────────────────┐
                    │ middleware / router │
                    └──────────┬──────────┘
                               │
                               ▼
                    ┌─────────────────────┐
                    │      Response       │
                    └──────────┬──────────┘
                               │
                               ▼
                    ┌─────────────────────┐
                    │ HTTP serialization  │
                    └──────────┬──────────┘
                               │
                               │ bytes
                               ▼
                    ┌─────────────────────┐
                    │      TCP Client     │
                    └─────────────────────┘
```

## Project Structure

```text
.
├── go.mod
├── main.go
├── public/
│   ├── index.html
│   └── static/
│       └── style.css
└── internal/
    └── httpserver/
        ├── server.go
        ├── request.go
        ├── response.go
        ├── router.go
        ├── middleware.go
        ├── static.go
        ├── request_test.go
        ├── response_test.go
        └── integration_test.go
```

The implementation stays in a single internal package because the project is intentionally small. Files are separated by responsibility without introducing additional package abstractions before they are needed.

## HTTP Framing

### Header Framing

HTTP headers are terminated by:

```text
\r\n\r\n
```

The server cannot assume that this sequence will be present in a single TCP read.

Incoming bytes are accumulated until the delimiter is found.

```text
Read #1 → "GET /hello HTTP"
Read #2 → "/1.1\r\nHost: loca"
Read #3 → "lhost:8080\r\n\r\n"
```

Only after the protocol delimiter is found does request parsing begin.

### Content-Length Bodies

For requests containing:

```text
Content-Length: N
```

the server continues reading until exactly `N` body bytes are available.

Bytes arriving after that boundary are preserved because they may belong to the next request on a persistent connection.

### Chunked Bodies

The server also understands:

```text
Transfer-Encoding: chunked
```

Example wire representation:

```text
5\r\n
hello\r\n
6\r\n
 world\r\n
0\r\n
\r\n
```

Chunk sizes are parsed as hexadecimal values. Chunk data is accumulated until the zero-sized terminating chunk is reached.

Trailer fields are consumed to preserve message framing but are currently ignored.

Requests containing both `Content-Length` and `Transfer-Encoding` are rejected to avoid ambiguous message framing.

## Persistent Connections

HTTP/1.1 connections are persistent by default.

Each TCP connection owns a `requestReader` containing:

```text
TCP connection
+
unconsumed buffered bytes
```

This is necessary because one TCP read may contain more than one request:

```text
[request 1][request 2]
```

After parsing request 1, its bytes are removed from the buffer while request 2 remains available for the next iteration.

`Connection: close` is also supported.

## Concurrency

Each accepted TCP connection is handled in its own goroutine:

```go
go handleConnection(...)
```

Connection-specific parser state is not shared between clients, so request processing currently requires no mutex.

This provides a simple goroutine-per-connection concurrency model without introducing worker pools or other concurrency abstractions that the project does not need.

## Timeouts and Limits

A client should not be able to consume server resources indefinitely by opening a connection and sending nothing or by sending an unbounded request.

The server therefore applies:

```text
Read timeout       5 seconds
Write timeout      5 seconds
Maximum headers    16 KB
Maximum body       1 MB
```

Oversized bodies are rejected before the declared body is read.

Oversized headers are rejected while the header block is still being accumulated rather than after an arbitrary amount of memory has already been consumed.

## Static File Security

Static resources are served from:

```text
public/static
```

Incoming paths are converted to filesystem paths and checked using `filepath.Rel`.

This prevents requests such as:

```text
/static/../../go.mod
```

from escaping the configured static root.

The implementation deliberately validates containment after filesystem path normalization rather than relying on a simple string check for `..`.

## Middleware

Handlers use a small function-based abstraction:

```go
type Handler func(Request) Response
type Middleware func(Handler) Handler
```

Request logging is implemented as middleware rather than being embedded in routing logic.

Logging can be enabled on Windows PowerShell with:

```powershell
$env:HTTP_SERVER_LOGGING="1"
go run .
```

To disable it:

```powershell
Remove-Item Env:HTTP_SERVER_LOGGING -ErrorAction SilentlyContinue
```

No interface-based or framework-style middleware system was introduced because a function-based handler chain is sufficient for the current requirements.

## Graceful Shutdown

`Ctrl+C` is converted into context cancellation using `signal.NotifyContext`.

During shutdown:

```text
interrupt received
      ↓
context cancelled
      ↓
listener closed
      ↓
new connections stop
      ↓
active connection goroutines finish
      ↓
server exits
```

A `sync.WaitGroup` tracks active connection handlers.

## Testing

Run the complete test suite with:

```powershell
go test ./...
```

Verbose output:

```powershell
go test -v ./...
```

The tests cover both individual parsing components and connection-level behavior.

Examples include:

- request-line parsing
- request-target validation
- header normalization
- duplicate `Content-Length` rejection
- response serialization
- chunked response serialization
- fragmented TCP reads
- fragmented request bodies
- multiple HTTP requests on one TCP connection
- request size limits
- chunked request decoding
- malformed chunk sizes
- ambiguous request framing

Connection-level tests use `net.Pipe`, allowing the TCP-facing parsing logic to be tested without opening real network ports.

## Running

Requires Go.

```powershell
git clone https://github.com/sbrsn97/go-http-from-scratch.git
cd go-http-from-scratch

go run .
```

The server listens on:

```text
http://localhost:8080
```

Build a binary:

```powershell
go build -o httpserver.exe .
.\httpserver.exe
```

Before committing changes:

```powershell
gofmt -w .
go vet ./...
go test ./...
go build ./...
```

## Benchmark

A local loopback load test was performed using Bombardier.

These results are machine- and environment-specific and should not be interpreted as production capacity measurements. Request logging was enabled during this baseline.

### `GET /hello`

| Connections | Requests | Avg Requests/sec | Avg Latency | Max Latency | Errors |
|---:|---:|---:|---:|---:|---:|
| 10 | 10,000 | 10,739 | 0.93 ms | 7.28 ms | 0 |
| 50 | 50,000 | 12,257 | 4.08 ms | 49.99 ms | 0 |
| 200 | 100,000 | 9,816 | 20.36 ms | 218.61 ms | 0 |

The throughput increase flattened between 50 and 200 concurrent connections while latency increased substantially, illustrating that increasing concurrency does not necessarily increase useful throughput.

### Additional Tests

| Endpoint | Connections | Requests | Avg Requests/sec | Avg Latency | Errors |
|---|---:|---:|---:|---:|---:|
| `GET /static/style.css` | 50 | 20,000 | 9,124 | 5.48 ms | 0 |
| `POST /echo` | 50 | 20,000 | 10,166 | 4.91 ms | 0 |

Static file serving performs an `os.ReadFile` for each request and is intentionally not backed by a cache.

No optimization was added solely to improve benchmark numbers.

## Design Decisions and Trade-offs

### Standard library first

The server is built using the Go standard library and raw `net.Conn` primitives rather than `net/http`.

The goal is protocol understanding, not replacing the standard HTTP implementation.

### Parse bytes before introducing abstractions

The project started with raw TCP reads and only introduced `Request`, `Response`, `requestReader`, routing, and middleware abstractions after their responsibilities became clear.

This avoided designing a framework before understanding the underlying problems.

### Connection-local state

Each connection owns its request buffer. This makes persistent connection handling straightforward and avoids synchronization for parser state.

### Small router

Routing is deliberately simple method/path branching rather than a trie, regex router, parameter framework, or reflection-based system.

Routing sophistication is outside the project's learning goals.

### Whole-body buffering

Decoded request bodies and generated response bodies are currently stored in memory.

Chunked transfer encoding is implemented to understand HTTP framing, not to provide a fully streaming server API.

A production implementation would likely expose reader/writer streaming abstractions instead.

### Static files are not cached

Files are read from disk on every request.

Caching would improve performance but would distract from the networking and HTTP goals of the project.

## Intentional Limitations

This is not a complete RFC-compatible or production-ready HTTP implementation.

Notable limitations include:

- HTTP/1.1 only
- no TLS / HTTPS
- no HTTP/2 or HTTP/3
- only origin-form request targets are supported
- no percent-decoding of request paths
- only `Transfer-Encoding: chunked` is supported
- chunk extensions are ignored
- trailer fields are consumed but not exposed to handlers
- request and response bodies are buffered in memory
- no streaming handler API
- no connection-count limit
- no sophisticated router
- no static file cache
- no compression
- no range requests
- no conditional requests
- no full RFC grammar implementation

These limitations are deliberate. The project focuses on the fundamentals of TCP, HTTP/1.1 framing, concurrency, resource limits, and Go rather than becoming a general-purpose web framework.

## What I Learned

The most important lesson from the project is that HTTP message boundaries do not come from TCP.

TCP provides an ordered byte stream. The application protocol is responsible for determining where one message ends and another begins.

That affects almost every part of an HTTP server:

```text
partial reads
persistent connections
Content-Length
chunked encoding
buffer management
request limits
timeouts
pipelining
```

The project also provided practical experience with Go concepts including:

```text
packages and modules
structs and methods
slices and byte buffers
multiple return values
error handling
defer
goroutines
channels through context cancellation
sync.WaitGroup
context
filesystem APIs
function-based middleware
unit testing
integration testing
```

Most importantly, the implementation evolved incrementally: functionality was built first, and abstractions were introduced when concrete duplication or responsibility boundaries appeared rather than pre-designing a framework.

## Non-Goals

This project is not intended to compete with Go's `net/http`, provide a production-ready HTTP stack, or maximize synthetic benchmark results.

Its purpose is to demonstrate understanding of the layers normally hidden behind a web framework.

---

Built as a hands-on exploration of HTTP/1.1, TCP networking, concurrency, and Go.