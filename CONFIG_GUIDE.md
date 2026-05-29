# T2D 配置指南（最佳实践）

本文档只保留对用户真正需要配置的字段说明。性能调优参数已经内置固定，避免误配置。

## 客户端必填/常用字段
- `mode`: 固定 `client`
- `listen_addr`: 本机 UDP 监听地址（WG Endpoint 指向这里）
- `server_upstream`: 服务端上行 TCP 地址
- `server_downstream`: 服务端下行 TCP 地址
- `upstream_crypto` / `downstream_crypto`: 上下行加密配置

常用非必填：
- `reconnect_interval`
- `reconnect_max_interval`
- `reconnect_jitter_ms`
- `session_timeout`
- `buffer_size`
- `max_packet_size`
- `log_level`

## 服务端必填/常用字段
- `mode`: 固定 `server`
- `upstream_port`: 上行 TCP 监听端口
- `downstream_port`: 下行 TCP 监听端口
- `backend_addr`: 后端 UDP 地址（通常 `127.0.0.1:51820`）
- `upstream_crypto` / `downstream_crypto`

常用非必填：
- `reconnect_interval`
- `session_timeout`
- `buffer_size`
- `max_packet_size`
- `log_level`

## 最小示例
请直接使用仓库根目录：
- `client_config.json`
- `server_config.json`

## 已内置固定（不允许用户配置）
- lane 数量
- 批量发送参数
- 上行 ring 队列容量
- 心跳间隔
- stripe/reorder 参数
- TCP/UDP socket buffer
- gnet buffer cap

## WireGuard 建议
```ini
MTU = 1420
```

## 测试方法
分开测试：
1. 基线（不加 netem）
2. 高延迟-only（`delay`）
3. 丢包-only（`loss`）

示例命令：
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
