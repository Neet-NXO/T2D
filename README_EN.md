# T2D-GNET

T2D-GNET is a `UDP in TCP` forwarder for UDP workloads (for example WireGuard), built on top of gnet with upstream/downstream split and multi-lane data paths.

## Goals
- Stable transport for UDP workloads in LAN and high-latency links.
- Simple operations: users only configure addresses/ports/crypto.
- Reproducible behavior: performance knobs are fixed in code, not user-tuned.

## Current Architecture
- Separate upstream/downstream TCP paths.
- Multi-lane connections on both client and server.
- Batched downstream writes on server.
- Performance-critical knobs are internal-only.

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

## Minimal Config Examples

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

## Internal Fixed Tuning (Not User Configurable)
The following are fixed in code and ignored from JSON:
- lane counts (up/down)
- upstream/downstream batch thresholds
- heartbeat interval
- upstream queue size
- upstream stripe/reorder flags and windows
- TCP/UDP socket buffers (16MB)
- gnet read/write buffer caps (1MB)

## WireGuard Notes
- Set explicit WG `MTU = 1420`.
- If extra encapsulations exist (PPPoE/VLAN/IPv6), start from `1380` or `1360`.

## Benchmarking Notes
Always separate scenarios:
- baseline (no netem)
- latency-only (delay only)
- loss-only (loss only)

```bash
# baseline
iperf3 -c 10.66.66.1 -P 1 -t 10

# latency-only
tc qdisc replace dev ens192 root netem delay 120ms limit 50000
iperf3 -c 10.66.66.1 -P 1 -t 10
tc qdisc del dev ens192 root

# loss-only
tc qdisc replace dev ens192 root netem loss 2%
iperf3 -c 10.66.66.1 -P 1 -t 10
tc qdisc del dev ens192 root
```

## Layout
- `cmd/t2d`: entrypoint
- `internal/client`: client path
- `internal/server`: server path
- `internal/protocol`: framing and control messages
- `internal/config`: config loading and defaults

## License
MIT
