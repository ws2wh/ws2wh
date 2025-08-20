# WS2WH ![build workflow](https://github.com/ws2wh/ws2wh/actions/workflows/build.yml/badge.svg) ![secscan workflow](https://github.com/ws2wh/ws2wh/actions/workflows/secscan.yml/badge.svg)

WS2WH is a lightweight bridge that connects WebSocket clients to HTTP webhook endpoints. It enables real-time,
bidirectional communication by converting WebSocket messages into HTTP POST requests and vice versa. This tool is
particularly useful when you need to integrate WebSocket-based clients with HTTP-only backend services, or when you
want to add WebSocket support to existing HTTP APIs without modifying the backend. With a simple configuration, WS2WH
handles the protocol translation and message routing, making it an ideal solution for scenarios requiring real-time
updates in HTTP-based architectures.

## Usage```shell
ws2wh -b https://example.com/api/v1/webhook -r /reply -l :3000 -p / -v INFO -h localhost
```

Parameters can be provided either as command-line flags or environment variables:

| Flag               | Environment Variable           | Default                   | Description                                                         |
| ------------------ | ------------------------------ | ------------------------- | ------------------------------------------------------------------- |
| `-b`               | `BACKEND_URL`                  | (required)                | Webhook backend URL that will receive POST requests from the relay  |
| `-r`               | `REPLY_PATH_PREFIX`            | `/reply`                  | Path prefix for backend replies                                     |
| `-l`               | `WS_PORT`                      | `:3000`                   | Address and port for the WebSocket server to listen on              |
| `-p`               | `WS_PATH`                      | `/`                       | Path where WebSocket connections will be upgraded                   |
| `-v`               | `LOG_LEVEL`                    | `INFO`                    | Log level (DEBUG, INFO, WARN, ERROR, OFF)                           |
| `-h`               | `REPLY_HOSTNAME` or `HOSTNAME` | `localhost`               | Hostname to use in reply channel                                    |
| `-metrics-enabled` | `METRICS_ENABLED`              | `false`                   | Enables Prometheus metrics endpoint                                 |
| `-metrics-port`    | `METRICS_PORT`                 | `9090`                    | Prometheus metrics port                                             |
| `-metrics-path`    | `METRICS_PATH`                 | `/metrics`                | Prometheus metrics path                                             |
| `-tls-enabled`     | `TLS_ENABLED`                  | `false`                   | Enables TLS                                                         |
| `-tls-cert-path`   | `TLS_CERT_PATH`                | (optional)                | TLS certificate path (PEM format). Required if TLS key path is set. |
| `-tls-key-path`    | `TLS_KEY_PATH`                 | (optional)                | TLS key path (PEM format). Required if TLS certificate path is set. |
| `-jwt-enabled`     | `JWT_ENABLED`                  | `false`                   | Enables JWT authentication                                          |
| `-jwt-secret-type` | `JWT_SECRET_TYPE`              | `jwks-url`                | JWT secret type (jwks-file, jwks-url, openid)                       |
| `-jwt-secret-path` | `JWT_SECRET_PATH`              | (required if JWT enabled) | Path to JWT secret (file path or URL depending on secret type)      |
| `-jwt-query-param` | `JWT_QUERY_PARAM`              | `token`                   | Query parameter name for JWT token                                  |
| `-jwt-issuer`      | `JWT_ISSUER`                   | (optional)                | JWT issuer                                                          |
| `-jwt-audience`    | `JWT_AUDIENCE`                 | (optional)                | JWT audience                                                        |

Example using environment variables:

```shell
export BACKEND_URL=https://example.com/api/v1/webhook
export REPLY_PATH_PREFIX=/reply
export WS_PORT=3000
export WS_PATH=/
export LOG_LEVEL=INFO
export REPLY_HOSTNAME=ws.example.com
export METRICS_ENABLED=true
export METRICS_PORT=9090
export METRICS_PATH=/metrics
export TLS_ENABLED=true
export TLS_CERT_PATH=./ws.example.com.crt
export TLS_KEY_PATH=./ws.example.com.key
export JWT_ENABLED=true
export JWT_SECRET_TYPE=jwks-url
export JWT_SECRET_PATH=https://your-domain/.well-known/jwks.json
export JWT_QUERY_PARAM=token
export JWT_ISSUER=https://your-issuer
export JWT_AUDIENCE=your-audience

ws2wh
```

## How it works

1. The WebSocket server listens for incoming connections on the specified address and port.
2. When a client connects, the server upgrades the connection to WebSocket and establishes a persistent connection.
3. The server converts WebSocket messages into HTTP POST requests and sends them to the specified backend URL.
4. The backend processes the request and sends a response back to the WebSocket server.
5. The server converts the response into a WebSocket message and sends it back to the client.
6. The process repeats for each message, allowing for real-time, bidirectional communication between the client and the
backend.

## Bridge to Backend Communication Protocol

### 1. WebSocket to Backend Messages

When WS2WH forwards messages to the backend, it sends HTTP POST requests with the following headers:

```http
Ws-Session-Id: <unique session identifier>
Ws-Query-String: <query string from the WS client (if any)>
Ws-Reply-Channel: <reply URL for this session>
Ws-Event: <event type>
Ws-Session-Jwt-Claims: <JSON string of JWT claims from the client (if any)>
```

Event types can be:

- `client-connected` - When a new WebSocket client connects
- `message-received` - When a WebSocket client sends a message
- `client-disconnected` - When a WebSocket client disconnects

The request body contains the raw message payload from the WebSocket client (empty for connection/disconnection events).

### 2. Backend to WebSocket Responses

The backend can respond in two ways:

#### 2.1 Immediate Response

Any response body in the 200-299 range will be forwarded back to the WebSocket client immediately.

#### 2.2 Async Reply

The backend can send messages later using the reply channel URL provided in `Ws-Reply-Channel` header:

```
POST <reply-channel-url>
Content-Type: text/plain

<message content>
```

### 3. Session Control

The backend can terminate a WebSocket session by including a special header in the reply:

```http
POST <reply-channel-url>
Ws-Command: terminate-session

<optional goodbye message>
```

This will:

1. Send the message body to the WebSocket client (if provided)
2. Close the WebSocket connection

### Example Flow

#### 1. WebSocket Client Connection

When a new WebSocket client connects to WS2WH:

```http
POST /webhook HTTP/1.1
Host: backend-server.com
Ws-Session-Id: 550e8400-e29b-41d4-a716-446655440000
Ws-Reply-Channel: http://ws2wh-host:3000/reply/550e8400-e29b-41d4-a716-446655440000
Ws-Event: client-connected
Content-Length: 0

```

#### 2. Client Message Forwarding

When the WebSocket client sends a message:

```http
POST /webhook HTTP/1.1
Host: backend-server.com
Ws-Session-Id: 550e8400-e29b-41d4-a716-446655440000
Ws-Reply-Channel: http://ws2wh-host:3000/reply/550e8400-e29b-41d4-a716-446655440000
Ws-Event: message-received
Content-Length: 13

Hello, backend
```

#### 3. Backend Responses

##### 3.1 Immediate Response

```http
HTTP/1.1 200 OK
Content-Type: text/plain
Content-Length: 24

Immediate response to client
```

##### 3.2 Async Response

```http
POST /reply/550e8400-e29b-41d4-a716-446655440000 HTTP/1.1
Host: ws2wh-host:3000
Content-Type: text/plain
Content-Length: 21

Async message to client
```

#### 4. Session Termination

The session can be proactively terminated by the backend by sending a `Ws-Command: terminate-session` header to a
session reply channel.

```http
POST /reply/550e8400-e29b-41d4-a716-446655440000 HTTP/1.1
Host: ws2wh-host:3000
Ws-Command: terminate-session
Content-Type: text/plain
Content-Length: 10

Goodbye!
```

In the example above, the backend server would send a `Ws-Command: terminate-session` header to a session reply
channel. The WebSocket session, along with the connection, will be closed gracefully with code 1000 by default.
Close code and reason can be customized by providing `Ws-Close-Code` and `Ws-Close-Reason` headers.

```http
POST /reply/550e8400-e29b-41d4-a716-446655440000 HTTP/1.1
Host: ws2wh-host:3000
Ws-Command: terminate-session
Ws-Close-Code: 1000
Ws-Close-Reason: Closing connection
```

`Ws-Close-Code` must be an integer between 1000 and 4999. Avoid reserved codes 1005, 1006, and 1015. Prefer
[standard close codes](https://www.rfc-editor.org/rfc/rfc6455#section-7.4); for application-defined purposes use the
3000–4999 range. `Ws-Close-Reason` is a UTF-8 string sent to the WebSocket client (keep it under 123 bytes to fit the
close frame payload limits).
