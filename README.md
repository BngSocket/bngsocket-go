# BngSocket

**BngSocket** is a powerful, session-based socket designed to enable bidirectional Remote Procedure Call (RPC) and reliable data transmission over stream protocols. It supports various stream-based protocols like TCP, TLS, Unix Domain Sockets, and Windows Named Pipes, while also providing the ability to transmit data via segmented channels.

## Features

- **Bidirectional RPC**: Both client and server can send and receive commands or data at any time.
- **Session-based**: BngSocket manages sessions, making it easy to maintain and track individual connections.
- **Segmented Channels**: Allows data transmission over channels, which can be assigned dynamically as needed.
- **Stream Protocols**: Exclusively supports stream-based protocols such as TCP, TLS, Unix Domain Sockets, and Windows Named Pipes.
- **Cross-Platform**: Works on Linux/Unix-based systems (using Unix Domain Sockets) and Windows (using Named Pipes).
