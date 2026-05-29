# T2D Configuration Guide

This guide covers only user-facing configuration. Performance-critical tuning is fixed in code.

## 1) Client Configuration

Required fields:
- `mode`: must be `client`
- `listen_addr`: local UDP listen address on the client
- `server_upstream`: upstream TCP endpoint on server side
- `server_downstream`: downstream TCP endpoint on server side
- `upstream_crypto`: upstream encryption config
- `downstream_crypto`: downstream encryption config

Common optional fields:
- `reconnect_interval`
- `reconnect_max_interval`
- `reconnect_jitter_ms`
- `session_timeout`
- `buffer_size`
- `max_packet_size`
- `log_level`

## 2) Server Configuration

Required fields:
- `mode`: must be `server`
- `upstream_port`: upstream TCP listen address (example `:9001`)
- `downstream_port`: downstream TCP listen address (example `:9002`)
- `backend_addr`: backend UDP address (for example `127.0.0.1:51820`)
- `upstream_crypto`: upstream decryption config (must match client upstream encryption)
- `downstream_crypto`: downstream encryption config (must match client downstream decryption)

Common optional fields:
- `reconnect_interval`
- `session_timeout`
- `buffer_size`
- `max_packet_size`
- `log_level`

## 3) Deployment Priorities

### Upstream/Downstream split
- Configure `server_upstream` and `server_downstream` separately.
- Listen on separate `upstream_port` and `downstream_port`.

### Split encryption
- Configure `upstream_crypto` and `downstream_crypto` independently.
- Different algorithms/passwords per direction are supported.
- Client and server must match on the same direction.

### IPv4/IPv6 directional isolation
- You can run upstream on IPv4 and downstream on IPv6.
- You can also route each direction via different ISPs or policy routes.

## 4) Minimal Examples

Use repository examples directly:
- `client_config.json`
- `server_config.json`

## 5) Fixed Internal Parameters

The following are not user-configurable:
- lane counts
- upstream/downstream batch parameters
- upstream queue size
- heartbeat interval
- stripe/reorder parameters
- TCP/UDP socket buffers
- gnet buffer caps

## 6) WireGuard Recommendation

Use explicit MTU:
```ini
MTU = 1420
```

If extra encapsulation exists (PPPoE/VLAN/IPv6), test `1380` or `1360`.

## 7) Basic Troubleshooting Checklist

- Cannot connect: verify both upstream and downstream ports are reachable.
- Only one-way traffic works: check downstream endpoint/port reachability first.
- Handshake then disconnect: verify direction-matched crypto settings.
- Throughput instability: avoid mixed test variables in one run.
