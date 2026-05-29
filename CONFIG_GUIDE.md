# T2D 配置指南

本文档只讲用户需要配置的内容。性能参数已经在代码内固定。

## 一、客户端配置

必填字段：
- `mode`: 固定为 `client`
- `listen_addr`: 客户端本地 UDP 监听地址（例如 WireGuard Endpoint 指向这里）
- `server_upstream`: 服务端上行 TCP 地址
- `server_downstream`: 服务端下行 TCP 地址
- `upstream_crypto`: 上行加密配置
- `downstream_crypto`: 下行加密配置

常用可选字段：
- `reconnect_interval`
- `reconnect_max_interval`
- `reconnect_jitter_ms`
- `session_timeout`
- `buffer_size`
- `max_packet_size`
- `log_level`

## 二、服务端配置

必填字段：
- `mode`: 固定为 `server`
- `upstream_port`: 上行 TCP 监听地址（示例 `:9001`）
- `downstream_port`: 下行 TCP 监听地址（示例 `:9002`）
- `backend_addr`: 后端 UDP 地址（例如 `127.0.0.1:51820`）
- `upstream_crypto`: 上行解密配置（需与客户端上行加密一致）
- `downstream_crypto`: 下行加密配置（需与客户端下行解密一致）

常用可选字段：
- `reconnect_interval`
- `session_timeout`
- `buffer_size`
- `max_packet_size`
- `log_level`

## 三、重点部署策略

### 1) 上下行分流
- `server_upstream` 和 `server_downstream` 必须分开配置。
- `upstream_port` 和 `downstream_port` 必须分开监听。

### 2) 分离式加密
- `upstream_crypto` 和 `downstream_crypto` 独立配置。
- 可以不同算法、不同密码，不需要强制一致。
- 但同一方向的客户端/服务端必须一致。

### 3) IPv4/IPv6 上下隔离
- 可将上行地址配置为 IPv4，下行地址配置为 IPv6。
- 也可按方向分别接入不同运营商或不同策略路由。

## 四、最小配置示例

请直接参考仓库文件：
- `client_config.json`
- `server_config.json`

## 五、固定内置参数（不支持用户配置）

以下参数由程序内部固定：
- lane 数量
- 上/下行批处理参数
- 上行队列容量
- 心跳间隔
- stripe/reorder 参数
- TCP/UDP socket buffer
- gnet buffer cap

## 六、WireGuard 建议

建议显式配置：
```ini
MTU = 1420
```

如果存在 PPPoE/VLAN/IPv6 叠加，可尝试 `1380` 或 `1360`。

## 七、基础排障清单

- 连接不上：先检查上下行端口是否都放通。
- 只能单向通信：优先检查 `server_downstream` / `downstream_port` 可达性。
- 握手后掉线：检查两端同方向加密配置是否完全一致。
- 性能波动：确认测试时没有混合多个网络变量（延迟、丢包、限速同时变化）。
