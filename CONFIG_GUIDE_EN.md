# T2D Configuration Guide (Best Practice)

This guide keeps only user-facing fields. Performance knobs are fixed in code to prevent misconfiguration.

## Client Required/Common Fields
Required:
- `mode` (`client`)
- `listen_addr`
- `server_upstream`
- `server_downstream`
- `upstream_crypto` / `downstream_crypto`

Common optional:
- `reconnect_interval`
- `reconnect_max_interval`
- `reconnect_jitter_ms`
- `session_timeout`
- `buffer_size`
- `max_packet_size`
- `log_level`

## Server Required/Common Fields
Required:
- `mode` (`server`)
- `upstream_port`
- `downstream_port`
- `backend_addr`
- `upstream_crypto` / `downstream_crypto`

Common optional:
- `reconnect_interval`
- `session_timeout`
- `buffer_size`
- `max_packet_size`
- `log_level`

## Minimal Examples
Use repository examples directly:
- `client_config.json`
- `server_config.json`

## Fixed Internals (Not User Configurable)
- lane counts
- batch thresholds
- upstream queue size
- heartbeat interval
- stripe/reorder settings
- TCP/UDP socket buffers
- gnet buffer caps

## WireGuard Recommendation
```ini
MTU = 1420
```

## Benchmark Method
Test separately:
1. baseline (no netem)
2. latency-only (`delay`)
3. loss-only (`loss`)

```bash
# baseline
iperf3 -c 10.66.66.1 -P 1 -t 10

# delay only
tc qdisc replace dev ens192 root netem delay 120ms limit 50000
iperf3 -c 10.66.66.1 -P 1 -t 10
tc qdisc del dev ens192 root

# loss only
tc qdisc replace dev ens192 root netem loss 2%
iperf3 -c 10.66.66.1 -P 1 -t 10
tc qdisc del dev ens192 root
```
