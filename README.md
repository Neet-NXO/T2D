# T2D-GNET

T2D-GNET 是一个面向 WireGuard 等 UDP 业务的 `UDP in TCP` 转发器，使用 gnet 实现上/下行分流与多连接数据面。

## 目标
- 在内网和跨区域环境下，稳定承载 UDP 业务（例如 WireGuard）。
- 保持部署简洁：用户只配置地址/端口/加密，不配置性能细节。
- 默认参数偏向可复现稳定性，不依赖用户手工调优。

## 当前设计要点
- 上行/下行 TCP 分流。
- 多 lane 连接（客户端与服务端各自维护）。
- 服务端下行批量发送。
- 关键性能参数改为代码内置固定值（不暴露给用户配置）。

## 快速开始

### 1. 构建
```bash
go build -o build/t2d ./cmd/t2d
```

### 2. 服务器启动
```bash
./build/t2d -config server_config.json
```

### 3. 客户端启动
```bash
./build/t2d -config client_config.json
```

## 最小配置示例

### 客户端
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

### 服务端
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

## 内置固定性能参数（非用户配置）
以下参数已在代码固定，配置文件不会覆盖：
- lane 数量（上行/下行）
- 上/下行批量发送阈值
- 心跳间隔
- 上行队列容量
- 上行 stripe/reorder 开关与窗口
- TCP/UDP socket buffer（16MB）
- gnet read/write buffer cap（1MB）

## WireGuard 使用建议
- WG 接口建议显式设置 `MTU = 1420`。
- 若链路叠加 PPPoE/VLAN/IPv6，可从 `1380` 或 `1360` 试探。

## 测试建议
分开测试，避免结论混淆：
- 基线：无 netem
- 高延迟-only：只加 `delay`
- 丢包-only：只加 `loss`

示例：
```bash
# 基线
iperf3 -c 10.66.66.1 -P 1 -t 10

# 高延迟-only
tc qdisc replace dev ens192 root netem delay 120ms limit 50000
iperf3 -c 10.66.66.1 -P 1 -t 10
tc qdisc del dev ens192 root

# 丢包-only
tc qdisc replace dev ens192 root netem loss 2%
iperf3 -c 10.66.66.1 -P 1 -t 10
tc qdisc del dev ens192 root
```

## 目录结构
- `cmd/t2d`: 程序入口
- `internal/client`: 客户端逻辑
- `internal/server`: 服务端逻辑
- `internal/protocol`: 协议编解码
- `internal/config`: 配置加载与默认值

## 许可证
MIT
