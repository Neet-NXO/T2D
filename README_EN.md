# T2D

T2D is a `UDP in TCP` forwarder for UDP workloads (WireGuard is a common case), built on top of gnet.

The key point is not just encapsulation. T2D splits the data plane into two independent paths:
- Upstream path: client UDP -> client upstream TCP -> server -> backend UDP
- Downstream path: backend UDP -> server downstream TCP -> client -> client UDP

This gives clear control boundaries for traffic policy, encryption, and IP planning.

## Design Priorities

- Upstream/downstream split: two independent TCP paths with independent reconnect, lane handling, and control messages.
- Directional split encryption: `upstream_crypto` and `downstream_crypto` are fully independent.
- IPv4/IPv6 directional isolation: upstream and downstream addresses are configured separately, so each direction can use different IP families or routes.
- Multi-lane data plane: default 4 lanes per direction, with `sessionID`-based lane mapping.
- Session consistency: UDP sessions are bound by `sessionID`; lane mapping is maintained with control frames (`hello/register/heartbeat`).
- Stable tuning defaults: performance-critical knobs are fixed in code to avoid unstable manual tuning.

## Implementation Characteristics

### 1) Real directional separation
Client keeps two distinct remote endpoints:
- upstream target: `server_upstream`
- downstream target: `server_downstream`

Server listens on two distinct ports:
- upstream listener: `upstream_port`
- downstream listener: `downstream_port`

This is physical connection separation, not logical multiplexing on one TCP connection.

### 2) Split encryption by direction
- Upstream: client encrypts, server decrypts (`upstream_crypto`)
- Downstream: server encrypts, client decrypts (`downstream_crypto`)

You can choose different algorithms and passwords per direction.

### 3) IPv4/IPv6 isolation by direction
Because upstream/downstream addresses are independent, you can deploy with:
- upstream on IPv4
- downstream on IPv6

You can also separate by ISP, DC egress, or policy route per direction.

Note: client session keys use `netip.AddrPort` with `Unmap`, avoiding duplicate sessions caused by IPv4 vs IPv4-mapped IPv6 representation differences.

## Quick Start

### 1. Build
```bash
go build -o build/t2d ./cmd/t2d
```

### 2. Start server
```bash
./build/t2d -config server_config.json
```

### 3. Start client
```bash
./build/t2d -config client_config.json
```

## Minimal Config

### Client
```json
{
  "mode": "client",
  "listen_addr": "0.0.0.0:51820",
  "server_upstream": "100.10.0.1:9001",
  "server_downstream": "100.10.0.1:9002",
  "reconnect_interval": 1,
  "reconnect_max_interval": 16,
  "reconnect_jitter_ms": 250,
  "session_timeout": 300,
  "buffer_size": 4194304,
  "max_packet_size": 1500,
  "log_level": "info",
  "upstream_crypto": {"type": "none", "password": ""},
  "downstream_crypto": {"type": "none", "password": ""}
}
```

### Server
```json
{
  "mode": "server",
  "upstream_port": ":9001",
  "downstream_port": ":9002",
  "backend_addr": "127.0.0.1:51820",
  "reconnect_interval": 5,
  "session_timeout": 300,
  "buffer_size": 4194304,
  "max_packet_size": 65536,
  "log_level": "info",
  "upstream_crypto": {"type": "none", "password": ""},
  "downstream_crypto": {"type": "none", "password": ""}
}
```

## Production Notes

- Apply separate ACL/rate-limit/audit rules to upstream/downstream ports.
- Use separate routing policies for upstream and downstream; IPv4/IPv6 split is supported by address design.
- Avoid `none` and `xor` in production.
- For WireGuard, set explicit `MTU = 1420`; if you have extra encapsulation (PPPoE/VLAN/IPv6), test `1380` or `1360`.

## Internal Fixed Parameters (Not User Configurable)

These are fixed in code:
- lane counts (up/down)
- batch thresholds (up/down)
- upstream queue size
- heartbeat interval
- upstream stripe/reorder settings
- TCP/UDP socket buffers (16MB)
- gnet read/write buffer caps (1MB)

## Repository Layout

- `cmd/t2d`: entrypoint
- `internal/client`: client data path
- `internal/server`: server data path
- `internal/protocol`: packet header and control frames
- `internal/crypto`: encryption implementation
- `internal/config`: config loading and defaults

## Related Docs

- `CONFIG_GUIDE_EN.md`: configuration guide
- `CRYPTO_README_EN.md`: encryption guide and troubleshooting
- `README.md`: Chinese README

## License

MIT
