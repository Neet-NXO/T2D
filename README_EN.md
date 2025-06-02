# T2D-GNET

A high-performance TCP-to-UDP proxy tool based on the gnet framework, supporting multiple encryption algorithms and flexible configuration options.

## Project Overview

T2D-GNET is a high-performance network proxy tool that primarily converts and proxies between TCP and UDP protocols. It uses the high-performance gnet network framework, supports multiple encryption algorithms, and can establish secure data transmission channels between clients and servers.

### Key Features

- 🚀 **High Performance**: Based on gnet framework, supports high-concurrency connections
- 🔐 **Bidirectional Independent Encryption**: Upstream and downstream data flows support different encryption algorithms and keys
- 🔀 **Upstream/Downstream Separation**: Independent upstream and downstream TCP connections for better performance and reliability
- 🛡️ **Multiple Encryption**: Supports AES-GCM, ChaCha20-Poly1305 and other encryption algorithms
- 🔧 **Flexible Configuration**: Rich configuration options, supports client and server modes
- 📊 **Session Management**: Intelligent UDP session management and timeout handling
- 🔄 **Auto Reconnection**: Supports automatic TCP connection reconnection on disconnection
- 📈 **Load Balancing**: Supports multiple load balancing strategies

## Architecture Design

### Core Architecture Features

**Upstream/Downstream Separation Design**: T2D-GNET uses independent upstream and downstream TCP connections instead of traditional single bidirectional connections. This design brings the following advantages:

- **Performance Optimization**: Upstream and downstream can be optimized independently, avoiding mutual interference
- **Reliability Enhancement**: Single-direction connection failure does not affect data transmission in the other direction
- **Flexible Configuration**: Different network parameters and routing can be configured for different directions

**Bidirectional Independent Encryption**: Supports upstream and downstream using completely different encryption algorithms and keys:

- **Enhanced Security**: Even if one direction's key is compromised, the other direction remains secure
- **Performance Balance**: Can choose the most suitable encryption algorithm based on data characteristics
- **Flexible Deployment**: Adapts to different security requirements and network environments

### Data Flow Diagram

```
Client Mode (Upstream/Downstream Separation):
                    ┌─ Upstream TCP Connection (Encryption A) ─┐
UDP App <--UDP--> T2D Client                                    T2D Server <--UDP--> Backend Service
                    └─ Downstream TCP Connection (Encryption B) ─┘

Server Mode:
                    ┌─ Upstream TCP Listen:9001 ─┐
Client                                              T2D Server <--UDP--> Backend UDP Service
                    └─ Downstream TCP Listen:9002 ─┘
```

## Quick Start

### Requirements

- Go 1.24.3 or higher
- Linux/macOS/Windows operating system

### Installation

1. Clone the project
```bash
git clone <repository-url>
cd T2D-GNET
```

2. Build the project
```bash
make build
```

Or use Go command directly:
```bash
go build -o build/t2d-gnet ./cmd/t2d-gnet
```

### Configuration Files

The project provides two example configuration files:

- `client_config.json` - Client configuration
- `server_config.json` - Server configuration

### Running

#### Start Server
```bash
make run-server
# or
./build/t2d-gnet -config server_config.json
```

#### Start Client
```bash
make run-client
# or
./build/t2d-gnet -config client_config.json
```

## Configuration

### Basic Configuration

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `mode` | string | Yes | Running mode: "client" or "server" |
| `log_level` | string | No | Log level: "debug", "info", "warn", "error" |
| `session_timeout` | int | No | UDP session timeout (seconds), default 300 |
| `reconnect_interval` | int | No | TCP reconnection interval (seconds), default 5 |

### Client Configuration

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `listen_addr` | string | Yes | UDP listen address, e.g. "0.0.0.0:51820" |
| `server_upstream` | string | Yes | Server upstream address, e.g. "192.168.1.100:9001" |
| `server_downstream` | string | Yes | Server downstream address, e.g. "192.168.1.100:9002" |

### Server Configuration

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `upstream_port` | string | Yes | Upstream listen port, e.g. ":9001" |
| `downstream_port` | string | Yes | Downstream listen port, e.g. ":9002" |
| `backend_addr` | string | Yes | Backend UDP address, e.g. "127.0.0.1:51820" |

### Encryption Configuration

#### Bidirectional Independent Encryption

One of T2D-GNET's core features is supporting completely independent encryption configurations for upstream and downstream data flows:

```json
{
  "upstream_crypto": {
    "type": "aes-256-gcm",
    "password": "upstream-secret-key-2024"
  },
  "downstream_crypto": {
    "type": "chacha20-poly1305",
    "password": "downstream-secret-key-2024"
  }
}
```

**Important Notes**:
- **Upstream Encryption**: Client → Server direction data encryption
- **Downstream Encryption**: Server → Client direction data encryption
- **Independent Keys**: Upstream and downstream can use completely different passwords and algorithms
- **Configuration Matching**: Client and server encryption configurations must match exactly

#### Encryption Configuration Examples

**High Security Configuration** (Different algorithms + Different keys):
```json
{
  "upstream_crypto": {
    "type": "aes-256-gcm",
    "password": "AES-upstream-key-32chars-long!!!"
  },
  "downstream_crypto": {
    "type": "xchacha20-poly1305",
    "password": "XChaCha20-downstream-key-secure"
  }
}
```

**Performance Optimized Configuration** (Fast algorithms):
```json
{
  "upstream_crypto": {
    "type": "chacha20-poly1305",
    "password": "fast-upstream-encryption-key"
  },
  "downstream_crypto": {
    "type": "chacha20-poly1305",
    "password": "fast-downstream-encryption-key"
  }
}
```

**No Encryption Configuration** (Plain text transmission):
```json
{
  "upstream_crypto": {
    "type": "none"
  },
  "downstream_crypto": {
    "type": "none"
  }
}
```

#### Supported Encryption Algorithms

- `none` - No encryption
- `xor` - XOR obfuscation
- `aes-128-gcm` - AES-128-GCM
- `aes-256-gcm` - AES-256-GCM
- `chacha20-poly1305` - ChaCha20-Poly1305
- `xchacha20-poly1305` - XChaCha20-Poly1305

### Performance Tuning Configuration

```json
{
  "multicore": true,
  "load_balancing": "least_connections",
  "socket_recv_buffer": 65536,
  "socket_send_buffer": 65536,
  "tcp_keep_alive": 30,
  "tcp_no_delay": true
}
```

## Use Cases

### 1. WireGuard Proxy

Transmit WireGuard UDP traffic through TCP, suitable for network environments that restrict UDP:

```bash
# Server side
./t2d-gnet -config server_config.json

# Client side
./t2d-gnet -config client_config.json

# WireGuard client connects to T2D client's listen address
wg-quick up wg0  # Configure endpoint to 127.0.0.1:51820
```

### 2. Game Acceleration

Provide TCP channel and encryption for UDP game traffic:

```json
{
  "mode": "client",
  "listen_addr": "0.0.0.0:7777",
  "server_upstream": "game-server.com:9001",
  "server_downstream": "game-server.com:9002",
  "upstream_crypto": {
    "type": "chacha20-poly1305",
    "password": "game-secret-key"
  }
}
```

### 3. DNS over TCP

Convert DNS UDP queries to TCP transmission:

```json
{
  "mode": "server",
  "upstream_port": ":5353",
  "downstream_port": ":5354",
  "backend_addr": "8.8.8.8:53"
}
```

## Development

### Project Structure

```
T2D-GNET/
├── cmd/t2d/           # Main program entry
├── internal/
│   ├── app/           # Application logic
│   ├── client/        # Client implementation
│   ├── server/        # Server implementation
│   ├── config/        # Configuration management
│   ├── crypto/        # Encryption module
│   ├── protocol/      # Protocol definition
│   ├── session/       # Session management
│   └── stats/         # Statistics
├── client_config.json # Client configuration example
├── server_config.json # Server configuration example
├── CONFIG_GUIDE.md    # Detailed configuration guide
├── CRYPTO_README.md   # Encryption feature documentation
└── Makefile          # Build script
```

### Available Make Commands

```bash
make build          # Build project
make clean          # Clean build files
make run-client     # Run client
make run-server     # Run server
make fmt            # Format code
make vet            # Code check
make test           # Run tests
make install        # Install to system
```

### Dependencies

- [gnet/v2](https://github.com/panjf2000/gnet) - High-performance network framework
- [golang.org/x/crypto](https://golang.org/x/crypto) - Go cryptography library

## Performance Features

- **High Concurrency**: Event-driven architecture based on gnet
- **Memory Optimization**: Zero-copy and object pool techniques
- **Load Balancing**: Supports multiple load balancing strategies
- **Connection Reuse**: TCP connection reuse and keep-alive
- **Buffer Optimization**: Configurable read/write buffers

## Security Features

- **End-to-End Encryption**: Supports multiple modern encryption algorithms
- **Key Management**: Secure key derivation and management
- **Session Isolation**: Independent session management and timeout
- **Anti-Replay**: Sequence number and timestamp verification

## Troubleshooting

### Common Issues

1. **Connection Failure**
   - Check firewall settings
   - Verify addresses and ports in configuration file
   - Confirm server is started

2. **Encryption Errors**
   - Ensure client and server use same encryption configuration
   - Check if passwords match
   - Verify encryption algorithm is supported

3. **Performance Issues**
   - Adjust buffer sizes
   - Enable multicore mode
   - Optimize load balancing strategy

### Log Analysis

Enable verbose logging:
```json
{
  "log_level": "debug",
  "log_path": "/var/log/t2d.log"
}
```

## Contributing

Welcome to submit Issues and Pull Requests!

## License

This project is licensed under the Apache License 2.0. See [LICENSE](LICENSE) file for details.

```
Copyright 2024 T2D-GNET Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

## More Documentation

- [Detailed Configuration Guide](CONFIG_GUIDE_EN.md)
- [Encryption Feature Documentation](CRYPTO_README_EN.md)