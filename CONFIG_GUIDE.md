# T2D 配置指南

## 概述

T2D 支持通过 JSON 配置文件进行详细配置。本文档详细说明了所有可用的配置选项。

## 配置文件结构

### 基础配置

#### `mode` (必需)
- **类型**: `string`
- **可选值**: `"client"` | `"server"`
- **说明**: 指定程序运行模式
  - `"client"`: 客户端模式，连接到远程服务器
  - `"server"`: 服务器模式，监听客户端连接
- **示例**: `"mode": "client"`

### 客户端配置 (mode = "client")

#### `listen_addr` (必需)
- **类型**: `string`
- **格式**: `"IP:端口"` 或 `":端口"`
- **说明**: 客户端UDP监听地址，用于接收本地应用的UDP连接
- **注意**: T2D 固定使用UDP协议，无需配置协议类型
- **示例**: 
  - `"listen_addr": "0.0.0.0:51820"` - 监听所有接口的 51820 端口
  - `"listen_addr": "127.0.0.1:51820"` - 仅监听本地回环接口
  - `"listen_addr": ":51820"` - 监听所有接口的 51820 端口（简写）

#### `server_upstream` (必需)
- **类型**: `string`
- **格式**: `"IP:端口"`
- **说明**: 服务器上行连接地址，客户端向此地址发送数据
- **示例**: `"server_upstream": "192.168.1.100:9001"`

#### `server_downstream` (必需)
- **类型**: `string`
- **格式**: `"IP:端口"`
- **说明**: 服务器下行连接地址，客户端从此地址接收数据
- **示例**: `"server_downstream": "192.168.1.100:9002"`

### 服务器配置 (mode = "server")

#### `upstream_port` (必需)
- **类型**: `string`
- **格式**: `":端口"` 或 `"IP:端口"`
- **说明**: 服务器上行监听端口，接收客户端发送的数据
- **示例**: 
  - `"upstream_port": ":9001"` - 监听所有接口的 9001 端口
  - `"upstream_port": "0.0.0.0:9001"` - 明确指定监听所有接口

#### `downstream_port` (必需)
- **类型**: `string`
- **格式**: `":端口"` 或 `"IP:端口"`
- **说明**: 服务器下行监听端口，向客户端发送数据
- **示例**: `"downstream_port": ":9002"`

#### `backend_addr` (必需)
- **类型**: `string`
- **格式**: `"IP:端口"`
- **说明**: 后端UDP服务地址，服务器将数据转发到此地址
- **注意**: T2D 固定使用UDP协议与后端通信
- **示例**: `"backend_addr": "127.0.0.1:51820"`

### 加密配置

#### `upstream_crypto` (可选)
- **类型**: `object`
- **说明**: 上行数据加密配置
- **默认值**: `{"type": "none"}`

#### `downstream_crypto` (可选)
- **类型**: `object`
- **说明**: 下行数据加密配置
- **默认值**: `{"type": "none"}`

##### 加密配置对象结构

###### `type` (必需)
- **类型**: `string`
- **可选值**: 
  - `"none"` - 无加密
  - `"xor"` - XOR 混淆
  - `"aes-128-gcm"` - AES-128-GCM 加密
  - `"aes-256-gcm"` - AES-256-GCM 加密
  - `"chacha20-poly1305"` - ChaCha20-Poly1305 加密
  - `"chacha20-ietf-poly1305"` - ChaCha20-IETF-Poly1305 加密
  - `"xchacha20-poly1305"` - XChaCha20-Poly1305 加密
  - `"xchacha20-ietf-poly1305"` - XChaCha20-IETF-Poly1305 加密
- **说明**: 指定使用的加密算法

###### `password` (条件必需)
- **类型**: `string`
- **说明**: 加密密码/密钥
- **必需条件**: 当 `type` 不为 `"none"` 或 `"xor"` 时必需
- **安全建议**: 
  - 使用至少 32 个字符的强密码
  - 包含大小写字母、数字和特殊字符
  - 避免使用常见词汇或个人信息
- **示例**: `"password": "MyVerySecurePassword2024!@#$"`

### 可选配置

#### `log_level` (可选)
- **类型**: `string`
- **可选值**: `"debug"` | `"info"` | `"warn"` | `"error"`
- **默认值**: `"info"`
- **说明**: 设置日志级别
- **示例**: `"log_level": "debug"`

#### `buffer_size` (可选)
- **类型**: `integer`
- **默认值**: `65536` (64KB)
- **说明**: 数据缓冲区大小（字节）
- **建议值**: 1024 - 1048576 (1KB - 1MB)
- **示例**: `"buffer_size": 32768`

#### `timeout` (可选)
- **类型**: `integer`
- **默认值**: `30`
- **单位**: 秒
- **说明**: 连接超时时间
- **示例**: `"timeout": 60`

#### `reconnect_interval` (可选，仅客户端)
- **类型**: `integer`
- **默认值**: `5`
- **单位**: 秒
- **说明**: 客户端重连间隔时间
- **建议**: 网络不稳定环境可适当增大
- **示例**: `"reconnect_interval": 10`

#### `session_timeout` (可选)
- **类型**: `integer`
- **默认值**: `300`
- **单位**: 秒
- **说明**: 会话超时时间，超过此时间未活动的连接将被关闭
- **建议**: 根据业务需求调整
- **示例**: `"session_timeout": 600`

#### `max_packet_size` (可选)
- **类型**: `integer`
- **默认值**: `65536`
- **单位**: 字节
- **说明**: 最大数据包大小限制
- **建议**: 根据网络MTU和业务需求调整
- **示例**: `"max_packet_size": 32768`

### gnet 网络引擎配置

#### `multicore` (可选)
- **类型**: `boolean`
- **默认值**: `true`
- **说明**: 是否启用多核模式，利用多个CPU核心处理网络事件
- **建议**: 多核服务器建议启用，单核或资源受限环境可关闭
- **示例**: `"multicore": true`

#### `lock_os_thread` (可选)
- **类型**: `boolean`
- **默认值**: `false`
- **说明**: 是否将goroutine锁定到OS线程，提高性能但增加资源消耗
- **建议**: 高性能要求时启用
- **示例**: `"lock_os_thread": false`

#### `load_balancing` (可选)
- **类型**: `string`
- **可选值**: `"round_robin"` | `"least_connections"` | `"source_addr_hash"`
- **默认值**: `"least_connections"`
- **说明**: 负载均衡算法
  - `"round_robin"`: 轮询算法
  - `"least_connections"`: 最少连接数算法
  - `"source_addr_hash"`: 源地址哈希算法
- **示例**: `"load_balancing": "least_connections"`

#### `num_event_loop` (可选)
- **类型**: `integer`
- **默认值**: `0`
- **说明**: 事件循环数量，0表示自动检测（通常等于CPU核心数）
- **建议**: 一般保持默认值，特殊情况下可手动指定
- **示例**: `"num_event_loop": 0`

#### `reuse_addr` (可选)
- **类型**: `boolean`
- **默认值**: `true`
- **说明**: 是否启用SO_REUSEADDR选项，允许地址重用
- **建议**: 通常保持启用
- **示例**: `"reuse_addr": true`

#### `reuse_port` (可选)
- **类型**: `boolean`
- **默认值**: `true`
- **说明**: 是否启用SO_REUSEPORT选项，允许多个进程绑定同一端口
- **建议**: Linux环境建议启用以提高性能
- **示例**: `"reuse_port": true`

#### `socket_recv_buffer` (可选)
- **类型**: `integer`
- **默认值**: `65536`
- **单位**: 字节
- **说明**: Socket接收缓冲区大小
- **建议**: 高吞吐量场景可适当增大
- **示例**: `"socket_recv_buffer": 65536`

#### `socket_send_buffer` (可选)
- **类型**: `integer`
- **默认值**: `65536`
- **单位**: 字节
- **说明**: Socket发送缓冲区大小
- **建议**: 高吞吐量场景可适当增大
- **示例**: `"socket_send_buffer": 65536`

#### `tcp_keep_alive` (可选)
- **类型**: `integer`
- **默认值**: `30`
- **单位**: 秒
- **说明**: TCP Keep-Alive间隔时间，0表示禁用
- **建议**: 长连接场景建议启用
- **示例**: `"tcp_keep_alive": 30`

#### `tcp_no_delay` (可选)
- **类型**: `boolean`
- **默认值**: `true`
- **说明**: 是否禁用Nagle算法，提高小包传输性能
- **建议**: 低延迟要求时启用
- **示例**: `"tcp_no_delay": true`

#### `ticker` (可选)
- **类型**: `boolean`
- **默认值**: `false`
- **说明**: 是否启用定时器功能
- **建议**: 需要定时任务时启用
- **示例**: `"ticker": false`

#### `read_buffer_cap` (可选)
- **类型**: `integer`
- **默认值**: `65536`
- **单位**: 字节
- **说明**: 读缓冲区容量
- **建议**: 根据数据包大小调整
- **示例**: `"read_buffer_cap": 65536`

#### `write_buffer_cap` (可选)
- **类型**: `integer`
- **默认值**: `65536`
- **单位**: 字节
- **说明**: 写缓冲区容量
- **建议**: 根据数据包大小调整
- **示例**: `"write_buffer_cap": 65536`

#### `edge_triggered_io` (可选)
- **类型**: `boolean`
- **默认值**: `false`
- **说明**: 是否启用边缘触发IO模式（仅Linux支持）
- **建议**: 高性能场景可启用，但需要更多CPU资源
- **示例**: `"edge_triggered_io": false`

#### `edge_triggered_io_chunk` (可选)
- **类型**: `integer`
- **默认值**: `8192`
- **单位**: 字节
- **说明**: 边缘触发IO模式下的数据块大小
- **前提**: 需要启用 `edge_triggered_io`
- **示例**: `"edge_triggered_io_chunk": 8192`

#### `bind_to_device` (可选)
- **类型**: `string`
- **默认值**: `""`
- **说明**: 绑定到特定网络设备（仅Linux支持）
- **示例**: `"bind_to_device": "eth0"`

#### `log_path` (可选)
- **类型**: `string`
- **默认值**: `""`
- **说明**: 日志文件路径，空字符串表示输出到控制台
- **示例**: `"log_path": "/var/log/T2D.log"`

## 完整配置示例

### 客户端配置示例

```json
{
  "mode": "client",
  "listen_addr": "0.0.0.0:51820",
  "server_upstream": "192.168.1.100:9001",
  "server_downstream": "192.168.1.100:9002",
  "reconnect_interval": 5,
  "session_timeout": 300,
  "buffer_size": 65536,
  "max_packet_size": 65536,
  "log_level": "info",
  "upstream_crypto": {
    "type": "aes-256-gcm",
    "password": "MyVerySecureUpstreamPassword2024!@#$"
  },
  "downstream_crypto": {
    "type": "chacha20-poly1305",
    "password": "MyVerySecureDownstreamPassword2024!@#$"
  },
  "multicore": true,
  "lock_os_thread": false,
  "load_balancing": "least_connections",
  "num_event_loop": 0,
  "reuse_addr": true,
  "reuse_port": true,
  "socket_recv_buffer": 65536,
  "socket_send_buffer": 65536,
  "tcp_keep_alive": 30,
  "tcp_no_delay": true,
  "ticker": false,
  "read_buffer_cap": 65536,
  "write_buffer_cap": 65536,
  "edge_triggered_io": false,
  "edge_triggered_io_chunk": 8192,
  "bind_to_device": "",
  "log_path": ""
}
```

### 服务器配置示例

```json
{
  "mode": "server",
  "upstream_port": ":9001",
  "downstream_port": ":9002",
  "backend_addr": "127.0.0.1:51820",
  "session_timeout": 300,
  "buffer_size": 65536,
  "max_packet_size": 65536,
  "log_level": "info",
  "upstream_crypto": {
    "type": "aes-256-gcm",
    "password": "MyVerySecureUpstreamPassword2024!@#$"
  },
  "downstream_crypto": {
    "type": "chacha20-poly1305",
    "password": "MyVerySecureDownstreamPassword2024!@#$"
  },
  "multicore": true,
  "lock_os_thread": false,
  "load_balancing": "least_connections",
  "num_event_loop": 0,
  "reuse_addr": true,
  "reuse_port": true,
  "socket_recv_buffer": 65536,
  "socket_send_buffer": 65536,
  "tcp_keep_alive": 30,
  "tcp_no_delay": true,
  "ticker": false,
  "read_buffer_cap": 65536,
  "write_buffer_cap": 65536,
  "edge_triggered_io": false,
  "edge_triggered_io_chunk": 8192,
  "bind_to_device": "",
  "log_path": ""
}
```

## 配置验证

### 必需字段检查

**客户端模式必需字段**:
- `mode`
- `listen_addr`
- `server_upstream`
- `server_downstream`

**服务器模式必需字段**:
- `mode`
- `upstream_port`
- `downstream_port`
- `backend_addr`

### 加密配置匹配

**重要**: 客户端和服务器的加密配置必须匹配：
- 客户端的 `upstream_crypto` = 服务器的 `upstream_crypto`
- 客户端的 `downstream_crypto` = 服务器的 `downstream_crypto`

### 端口配置注意事项

1. **端口范围**: 1-65535
2. **权限**: 使用 1024 以下端口需要管理员权限
3. **冲突**: 确保端口未被其他程序占用
4. **防火墙**: 确保相关端口在防火墙中开放

## 常见配置错误

### 1. 加密配置不匹配
```json
// 错误：客户端和服务器加密配置不一致
// 客户端
"upstream_crypto": {"type": "aes-256-gcm", "password": "pass1"}
// 服务器
"upstream_crypto": {"type": "aes-128-gcm", "password": "pass2"}
```

### 2. 缺少必需的密码
```json
// 错误：使用加密算法但未提供密码
"upstream_crypto": {
  "type": "aes-256-gcm"
  // 缺少 "password" 字段
}
```

### 3. 无效的地址格式
```json
// 错误：地址格式不正确
"listen_addr": "51820",        // 缺少 IP 部分
"server_upstream": "192.168.1.100", // 缺少端口部分
"backend_addr": ":51820"      // 缺少 IP 部分
```

### 4. 端口冲突
```json
// 错误：上行和下行使用相同端口
"upstream_port": ":9001",
"downstream_port": ":9001"  // 与上行端口冲突
```

## 性能调优建议

### 缓冲区大小
- **小文件传输**: 8KB - 16KB
- **大文件传输**: 64KB - 256KB
- **高并发场景**: 32KB - 64KB

### 加密算法选择
- **高安全性**: `aes-256-gcm`, `xchacha20-poly1305`
- **高性能**: `aes-128-gcm`, `chacha20-poly1305`
- **移动设备**: `chacha20-poly1305`, `xchacha20-poly1305`

### 超时设置
- **本地网络**: 10-30 秒
- **互联网**: 30-60 秒
- **不稳定网络**: 60-120 秒


## 配置文件位置

**指定配置文件**:
```bash
./t2d -config /path/to/your/config.json
```