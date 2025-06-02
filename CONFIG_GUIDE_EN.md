# T2D-GNET Configuration Guide

This document provides a comprehensive guide to configuring T2D-GNET, including all available configuration options, examples, and best practices.

## Configuration File Format

T2D-GNET uses JSON format configuration files. The configuration structure varies depending on the operating mode (client or server).

## Basic Configuration Structure

### Required Fields

```json
{
  "mode": "client|server"
}
```

### Complete Configuration Template

```json
{
  "mode": "client",
  "log_level": "info",
  "log_path": "/var/log/t2d.log",
  "session_timeout": 300,
  "reconnect_interval": 5,
  "buffer_size": 65536,
  
  // Client-specific configuration
  "listen_addr": "0.0.0.0:51820",
  "server_upstream": "192.168.1.100:9001",
  "server_downstream": "192.168.1.100:9002",
  
  // Server-specific configuration
  "upstream_port": ":9001",
  "downstream_port": ":9002",
  "backend_addr": "127.0.0.1:51820",
  
  // Encryption configuration
  "upstream_crypto": {
    "type": "aes-256-gcm",
    "password": "upstream-secret-key"
  },
  "downstream_crypto": {
    "type": "chacha20-poly1305",
    "password": "downstream-secret-key"
  },
  
  // Performance tuning
  "multicore": true,
  "load_balancing": "least_connections",
  "socket_recv_buffer": 65536,
  "socket_send_buffer": 65536,
  "tcp_keep_alive": 30,
  "tcp_no_delay": true
}
```

## Configuration Fields Reference

### Basic Configuration

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `mode` | string | Yes | - | Operating mode: "client" or "server" |
| `log_level` | string | No | "info" | Log level: "debug", "info", "warn", "error" |
| `log_path` | string | No | stdout | Log file path, empty for stdout |
| `session_timeout` | int | No | 300 | UDP session timeout in seconds |
| `reconnect_interval` | int | No | 5 | TCP reconnection interval in seconds |
| `buffer_size` | int | No | 65536 | Default buffer size in bytes |

### Client Configuration

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `listen_addr` | string | Yes | - | UDP listen address and port |
| `server_upstream` | string | Yes | - | Server upstream TCP address |
| `server_downstream` | string | Yes | - | Server downstream TCP address |

#### Client Configuration Examples

**Basic Client Configuration**:
```json
{
  "mode": "client",
  "listen_addr": "0.0.0.0:51820",
  "server_upstream": "vpn.example.com:9001",
  "server_downstream": "vpn.example.com:9002"
}
```

**Client with Encryption**:
```json
{
  "mode": "client",
  "listen_addr": "127.0.0.1:51820",
  "server_upstream": "secure.example.com:9001",
  "server_downstream": "secure.example.com:9002",
  "upstream_crypto": {
    "type": "aes-256-gcm",
    "password": "client-upstream-secret-2024"
  },
  "downstream_crypto": {
    "type": "chacha20-poly1305",
    "password": "client-downstream-secret-2024"
  }
}
```

### Server Configuration

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `upstream_port` | string | Yes | - | Upstream TCP listen port |
| `downstream_port` | string | Yes | - | Downstream TCP listen port |
| `backend_addr` | string | Yes | - | Backend UDP service address |

#### Server Configuration Examples

**Basic Server Configuration**:
```json
{
  "mode": "server",
  "upstream_port": ":9001",
  "downstream_port": ":9002",
  "backend_addr": "127.0.0.1:51820"
}
```

**Server with Encryption**:
```json
{
  "mode": "server",
  "upstream_port": ":9001",
  "downstream_port": ":9002",
  "backend_addr": "10.0.0.1:53",
  "upstream_crypto": {
    "type": "aes-256-gcm",
    "password": "server-upstream-secret-2024"
  },
  "downstream_crypto": {
    "type": "chacha20-poly1305",
    "password": "server-downstream-secret-2024"
  }
}
```

### Encryption Configuration

#### Upstream Crypto Configuration

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `type` | string | Yes | - | Encryption algorithm type |
| `password` | string | Conditional | - | Encryption password (required unless type is "none") |

#### Downstream Crypto Configuration

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `type` | string | Yes | - | Encryption algorithm type |
| `password` | string | Conditional | - | Encryption password (required unless type is "none") |

#### Supported Encryption Types

- `none` - No encryption
- `xor` - XOR obfuscation
- `aes-128-gcm` - AES-128-GCM
- `aes-256-gcm` - AES-256-GCM
- `chacha20-poly1305` - ChaCha20-Poly1305
- `xchacha20-poly1305` - XChaCha20-Poly1305

#### Encryption Configuration Examples

**No Encryption**:
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

**Symmetric Encryption**:
```json
{
  "upstream_crypto": {
    "type": "aes-256-gcm",
    "password": "shared-secret-key-2024"
  },
  "downstream_crypto": {
    "type": "aes-256-gcm",
    "password": "shared-secret-key-2024"
  }
}
```

**Asymmetric Encryption**:
```json
{
  "upstream_crypto": {
    "type": "aes-256-gcm",
    "password": "upstream-only-secret-key"
  },
  "downstream_crypto": {
    "type": "xchacha20-poly1305",
    "password": "downstream-only-secret-key"
  }
}
```

### Performance Configuration

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `multicore` | bool | No | false | Enable multicore processing |
| `load_balancing` | string | No | "round_robin" | Load balancing strategy |
| `socket_recv_buffer` | int | No | 65536 | Socket receive buffer size |
| `socket_send_buffer` | int | No | 65536 | Socket send buffer size |
| `tcp_keep_alive` | int | No | 30 | TCP keep-alive interval in seconds |
| `tcp_no_delay` | bool | No | true | Enable TCP_NODELAY option |

#### Load Balancing Strategies

- `round_robin` - Round-robin distribution
- `least_connections` - Least connections first
- `random` - Random selection
- `hash` - Hash-based distribution

#### Performance Configuration Example

```json
{
  "mode": "server",
  "upstream_port": ":9001",
  "downstream_port": ":9002",
  "backend_addr": "127.0.0.1:51820",
  
  "multicore": true,
  "load_balancing": "least_connections",
  "socket_recv_buffer": 131072,
  "socket_send_buffer": 131072,
  "tcp_keep_alive": 60,
  "tcp_no_delay": true,
  
  "session_timeout": 600,
  "reconnect_interval": 3,
  "buffer_size": 131072
}
```

## Complete Configuration Examples

### Example 1: WireGuard Proxy

**Client Configuration** (`client_wg.json`):
```json
{
  "mode": "client",
  "log_level": "info",
  "listen_addr": "127.0.0.1:51820",
  "server_upstream": "vpn.example.com:9001",
  "server_downstream": "vpn.example.com:9002",
  "session_timeout": 300,
  "reconnect_interval": 5,
  
  "upstream_crypto": {
    "type": "chacha20-poly1305",
    "password": "wireguard-upstream-key-2024"
  },
  "downstream_crypto": {
    "type": "chacha20-poly1305",
    "password": "wireguard-downstream-key-2024"
  }
}
```

**Server Configuration** (`server_wg.json`):
```json
{
  "mode": "server",
  "log_level": "info",
  "upstream_port": ":9001",
  "downstream_port": ":9002",
  "backend_addr": "127.0.0.1:51820",
  "session_timeout": 300,
  
  "upstream_crypto": {
    "type": "chacha20-poly1305",
    "password": "wireguard-upstream-key-2024"
  },
  "downstream_crypto": {
    "type": "chacha20-poly1305",
    "password": "wireguard-downstream-key-2024"
  },
  
  "multicore": true,
  "load_balancing": "least_connections"
}
```

### Example 2: High-Performance Gaming

**Client Configuration** (`client_gaming.json`):
```json
{
  "mode": "client",
  "log_level": "warn",
  "listen_addr": "0.0.0.0:7777",
  "server_upstream": "game-proxy.example.com:9001",
  "server_downstream": "game-proxy.example.com:9002",
  "session_timeout": 120,
  "reconnect_interval": 2,
  "buffer_size": 32768,
  
  "upstream_crypto": {
    "type": "chacha20-poly1305",
    "password": "gaming-fast-encryption-key"
  },
  "downstream_crypto": {
    "type": "chacha20-poly1305",
    "password": "gaming-fast-encryption-key"
  },
  
  "tcp_no_delay": true,
  "socket_recv_buffer": 32768,
  "socket_send_buffer": 32768
}
```

### Example 3: DNS over TCP

**Server Configuration** (`server_dns.json`):
```json
{
  "mode": "server",
  "log_level": "info",
  "upstream_port": ":5353",
  "downstream_port": ":5354",
  "backend_addr": "8.8.8.8:53",
  "session_timeout": 60,
  
  "upstream_crypto": {
    "type": "aes-128-gcm",
    "password": "dns-proxy-encryption-key"
  },
  "downstream_crypto": {
    "type": "aes-128-gcm",
    "password": "dns-proxy-encryption-key"
  },
  
  "multicore": false,
  "load_balancing": "round_robin",
  "buffer_size": 4096
}
```

### Example 4: Development/Testing

**Client Configuration** (`client_dev.json`):
```json
{
  "mode": "client",
  "log_level": "debug",
  "listen_addr": "127.0.0.1:8080",
  "server_upstream": "localhost:9001",
  "server_downstream": "localhost:9002",
  "session_timeout": 60,
  "reconnect_interval": 1,
  
  "upstream_crypto": {
    "type": "none"
  },
  "downstream_crypto": {
    "type": "none"
  }
}
```

**Server Configuration** (`server_dev.json`):
```json
{
  "mode": "server",
  "log_level": "debug",
  "upstream_port": ":9001",
  "downstream_port": ":9002",
  "backend_addr": "127.0.0.1:8081",
  "session_timeout": 60,
  
  "upstream_crypto": {
    "type": "none"
  },
  "downstream_crypto": {
    "type": "none"
  }
}
```

## Configuration Validation

### Required Field Validation

T2D-GNET validates the following requirements:

1. **Mode Field**: Must be either "client" or "server"
2. **Client Mode**: Requires `listen_addr`, `server_upstream`, `server_downstream`
3. **Server Mode**: Requires `upstream_port`, `downstream_port`, `backend_addr`
4. **Encryption**: If crypto type is not "none", password is required

### Common Validation Errors

```bash
# Missing required fields
Error: missing required field 'listen_addr' for client mode

# Invalid encryption type
Error: unsupported encryption type 'invalid-cipher'

# Invalid address format
Error: invalid address format '192.168.1.100'

# Port conflicts
Error: upstream and downstream ports cannot be the same
```

## Environment Variables

T2D-GNET supports environment variable substitution in configuration files:

```json
{
  "mode": "client",
  "listen_addr": "${LISTEN_ADDR:-0.0.0.0:51820}",
  "server_upstream": "${SERVER_HOST}:${UPSTREAM_PORT}",
  "server_downstream": "${SERVER_HOST}:${DOWNSTREAM_PORT}",
  
  "upstream_crypto": {
    "type": "aes-256-gcm",
    "password": "${UPSTREAM_PASSWORD}"
  },
  "downstream_crypto": {
    "type": "aes-256-gcm",
    "password": "${DOWNSTREAM_PASSWORD}"
  }
}
```

### Environment Variable Usage

```bash
# Set environment variables
export LISTEN_ADDR="0.0.0.0:51820"
export SERVER_HOST="vpn.example.com"
export UPSTREAM_PORT="9001"
export DOWNSTREAM_PORT="9002"
export UPSTREAM_PASSWORD="secure-upstream-key"
export DOWNSTREAM_PASSWORD="secure-downstream-key"

# Run with environment variables
./t2d-gnet -config client_config.json
```

## Configuration Best Practices

### 1. Security Best Practices

- **Strong Passwords**: Use long, random passwords for encryption
- **Different Keys**: Use different passwords for upstream and downstream
- **File Permissions**: Set restrictive permissions on configuration files
- **Environment Variables**: Use environment variables for sensitive data

```bash
# Set secure file permissions
chmod 600 client_config.json server_config.json

# Use environment variables for passwords
export T2D_UPSTREAM_KEY="$(openssl rand -base64 32)"
export T2D_DOWNSTREAM_KEY="$(openssl rand -base64 32)"
```

### 2. Performance Best Practices

- **Buffer Sizes**: Adjust buffer sizes based on expected traffic
- **Multicore**: Enable multicore for high-traffic scenarios
- **Keep-Alive**: Tune TCP keep-alive for network conditions
- **Session Timeout**: Set appropriate session timeouts

### 3. Monitoring and Logging

```json
{
  "log_level": "info",
  "log_path": "/var/log/t2d/t2d.log",
  "stats_interval": 60,
  "health_check_port": ":8080"
}
```

### 4. High Availability Configuration

```json
{
  "mode": "client",
  "listen_addr": "0.0.0.0:51820",
  
  "servers": [
    {
      "upstream": "primary.example.com:9001",
      "downstream": "primary.example.com:9002",
      "priority": 1
    },
    {
      "upstream": "backup.example.com:9001",
      "downstream": "backup.example.com:9002",
      "priority": 2
    }
  ],
  
  "failover_timeout": 10,
  "health_check_interval": 30
}
```

## Troubleshooting Configuration Issues

### 1. Configuration File Syntax

```bash
# Validate JSON syntax
jq . client_config.json

# Check for common issues
./t2d-gnet -config client_config.json -validate
```

### 2. Network Configuration

```bash
# Test connectivity
telnet server.example.com 9001
telnet server.example.com 9002

# Check firewall rules
sudo iptables -L
sudo ufw status
```

### 3. Encryption Configuration

```bash
# Test with no encryption first
{
  "upstream_crypto": {"type": "none"},
  "downstream_crypto": {"type": "none"}
}

# Enable debug logging
{
  "log_level": "debug",
  "crypto_debug": true
}
```

## Migration and Upgrades

### Configuration Migration

When upgrading T2D-GNET versions, check for configuration changes:

```bash
# Backup current configuration
cp client_config.json client_config.json.backup

# Validate new configuration format
./t2d-gnet-new -config client_config.json -validate

# Update configuration if needed
./t2d-gnet-new -config client_config.json -migrate
```

### Gradual Deployment

1. **Test Environment**: Deploy and test new configuration
2. **Staging Environment**: Validate with production-like traffic
3. **Production Rollout**: Deploy with monitoring and rollback plan

---

For more information about encryption features, see [CRYPTO_README_EN.md](CRYPTO_README_EN.md).