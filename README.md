# WS2WH

WS2WH is a lightweight bridge that connects WebSocket clients to HTTP webhook endpoints. It enables real-time, bidirectional communication by converting WebSocket messages into HTTP POST requests and vice versa. This tool is particularly useful when you need to integrate WebSocket-based clients with HTTP-only backend services, or when you want to add WebSocket support to existing HTTP APIs without modifying the backend. With a simple configuration, WS2WH handles the protocol translation and message routing, making it an ideal solution for scenarios requiring real-time updates in HTTP-based architectures.

## Usage

```
ws2wh -b https://example.com/api/v1/webhook -r /reply -l :3000 -p /
```

Parameters:
- `-b` (required) - Webhook backend URL that will receive POST requests from the relay
- `-r` (default: "/reply") - Path prefix for backend replies
- `-l` (default: ":3000") - Address and port for the WebSocket server to listen on
- `-p` (default: "/") - Path where WebSocket connections will be upgraded

## How it works

1. The WebSocket server listens for incoming connections on the specified address and port.
2. When a client connects, the server upgrades the connection to WebSocket and establishes a persistent connection.
3. The server converts WebSocket messages into HTTP POST requests and sends them to the specified backend URL.
4. The backend processes the request and sends a response back to the WebSocket server.
5. The server converts the response into a WebSocket message and sends it back to the client.
6. The process repeats for each message, allowing for real-time, bidirectional communication between the client and the backend.

## Bridge to Backend Communication Protocol

### 1. WebSocket to Backend Messages

When WS2WH forwards messages to the backend, it sends HTTP POST requests with the following headers:

```
Ws-Session-Id: <unique session identifier>
Ws-Reply-Channel: <reply URL for this session>
Ws-Event: <event type>
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

```
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

##### 4.1 With Goodbye Message
```http
POST /reply/550e8400-e29b-41d4-a716-446655440000 HTTP/1.1
Host: ws2wh-host:3000
Content-Type: text/plain
Ws-Command: terminate-session
Content-Length: 14

Goodbye client
```

##### 4.2 Client Disconnection Notification
```http
POST /webhook HTTP/1.1
Host: backend-server.com
Ws-Session-Id: 550e8400-e29b-41d4-a716-446655440000
Ws-Reply-Channel: http://ws2wh-host:3000/reply/550e8400-e29b-41d4-a716-446655440000
Ws-Event: client-disconnected
Content-Length: 0

```
