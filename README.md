# T2D

T2D 是一个面向 UDP 业务（典型场景是 WireGuard）的 `UDP in TCP` 转发器，基于 gnet 实现。

它的核心不是“把 UDP 套进 TCP”这么简单，而是把数据面拆成两条独立链路：
- 上行链路：客户端 UDP -> 客户端上行 TCP -> 服务端 -> 后端 UDP
- 下行链路：后端 UDP -> 服务端下行 TCP -> 客户端 -> 客户端 UDP

这使得 T2D 在链路治理、加密策略和 IP 规划上有明确的可控边界。

## 设计重点

- 上下行分流：上行和下行是两条独立 TCP 通道，独立重连、独立 lane、独立控制帧。
- 分离式加密：`upstream_crypto` 和 `downstream_crypto` 完全独立，可不同算法、不同密码。
- IPv4/IPv6 上下隔离：上行地址与下行地址分开配置，可按方向拆分到 IPv4/IPv6 或不同网络路径。
- 多 lane 数据面：默认上/下行各 4 lane，按 `sessionID` 映射 lane，减少单连接拥塞影响。
- 会话一致性：UDP 会话由 `sessionID` 绑定，lane 通过控制帧 (`hello/register/heartbeat`) 维护映射。
- 参数稳态：性能关键参数固定在代码内，避免“手调参数后性能随机波动”。

## 程序特点（结合实现）

### 1) 上下行是两条真正独立的数据路径
客户端同时维护：
- 上行目标：`server_upstream`
- 下行目标：`server_downstream`

服务端同时监听：
- 上行端口：`upstream_port`
- 下行端口：`downstream_port`

这不是同一连接上的逻辑复用，而是物理分离的 TCP 通道。

### 2) 分离式加密是“一方向一策略”
- 上行：客户端加密 -> 服务端解密（`upstream_crypto`）
- 下行：服务端加密 -> 客户端解密（`downstream_crypto`）

你可以配置为：
- 上行 `aes-256-gcm`，下行 `xchacha20-poly1305`
- 或上行加密、下行 `none`
- 或两边都同算法但不同密码

只要两端同方向配置一致即可。

### 3) IPv4/IPv6 可按方向隔离部署
因为上下行地址完全分离，你可以做：
- 上行走 IPv4 公网地址
- 下行走 IPv6 地址

同样也可以上下行分别走不同 IDC 出口、不同运营商或不同策略路由。

补充：客户端会话键使用 `netip.AddrPort` 且对地址做 `Unmap` 归一化，避免 IPv4 与 IPv4-mapped IPv6 表示差异带来的重复会话。

## 快速开始

### 1. 构建
```bash
go build -o build/t2d ./cmd/t2d
```

### 2. 启动服务端
```bash
./build/t2d -config server_config.json
```

### 3. 启动客户端
```bash
./build/t2d -config client_config.json
```

## 最小配置

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

## 生产部署建议

- 把上下行端口分开做 ACL、限速、审计。
- 把上下行地址分开做路由，必要时使用不同 IP 协议族（IPv4/IPv6）。
- 生产环境避免 `none` / `xor`。
- WireGuard 建议显式设置 `MTU = 1420`；若叠加 PPPoE/VLAN/IPv6，可试 `1380` 或 `1360`。

## 内置固定参数（不开放配置）

以下参数由代码固定：
- 上/下行 lane 数量
- 上/下行批处理阈值
- 上行队列容量
- 心跳间隔
- 上行 stripe/reorder 参数
- TCP/UDP socket buffer（16MB）
- gnet read/write buffer cap（1MB）

## 目录结构

- `cmd/t2d`: 程序入口
- `internal/client`: 客户端数据面
- `internal/server`: 服务端数据面
- `internal/protocol`: 包头与控制帧
- `internal/crypto`: 加密实现
- `internal/config`: 配置加载与默认值

## 相关文档

- `CONFIG_GUIDE.md`: 配置项说明
- `CRYPTO_README.md`: 加密配置与排障
- `README_EN.md`: English README

## 许可证

MIT
