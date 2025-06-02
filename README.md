# T2D

一个基于 gnet 框架的高性能 UDP Over TCP 转发工具，支持多种加密算法和灵活的配置选项。

## 项目简介

T2D 是一个高性能的网络代理工具，主要功能是在 TCP 和 UDP 协议之间进行转换和代理。它采用了高性能的 gnet 网络框架，支持多种加密算法，可以在客户端和服务器之间建立安全的数据传输通道。

### 主要特性

- 🚀 **高性能**: 基于 gnet 框架，支持高并发连接
- 🔐 **双向独立加密**: 上行和下行数据流支持不同的加密算法和密钥
- 🔀 **上下行分离**: 独立的上行和下行 TCP 连接，提供更好的性能和可靠性
- 🛡️ **多种加密**: 支持 AES-GCM、ChaCha20-Poly1305 等多种加密算法
- 🔧 **灵活配置**: 丰富的配置选项，支持客户端和服务器模式
- 📊 **会话管理**: 智能的 UDP 会话管理和超时处理
- 🔄 **自动重连**: 支持 TCP 连接断线自动重连
- 📈 **负载均衡**: 支持多种负载均衡策略

## 架构设计

### 核心架构特性

**上下行分离设计**: T2D 采用独立的上行和下行 TCP 连接，而不是传统的单一双向连接。这种设计带来以下优势：

- **性能优化**: 上行和下行可以独立优化，避免相互干扰
- **灵活配置**: 可以为不同方向配置不同的网络参数和路由

**双向独立加密**: 支持上行和下行使用完全不同的加密算法和密钥：

- **安全性增强**: 即使一个方向的密钥泄露，另一个方向仍然安全
- **性能平衡**: 可以根据数据特性选择最适合的加密算法
- **灵活部署**: 适应不同的安全需求和网络环境

### 数据流图

```
客户端模式 (上下行分离):
                    ┌─ 上行TCP连接(加密A) ─┐
UDP应用 <--UDP--> T2D客户端                    T2D服务器 <--UDP--> 后端服务
                    └─ 下行TCP连接(加密B) ─┘

服务器模式:
                    ┌─ 上行TCP监听:9001(加密A) ─┐
客户端                                           T2D服务器 <--UDP--> 后端UDP服务
                    └─ 下行TCP监听:9002(加密B) ─┘
```

## 快速开始

### 环境要求

- Go 1.24.3 或更高版本
- Linux/macOS/Windows 操作系统

### 安装

1. 克隆项目
```bash
git clone <repository-url>
cd T2D
```

2. 构建项目
```bash
make build
```

或者直接使用 Go 命令：
```bash
go build -o build/T2D ./cmd/T2D
```

### 配置文件

项目提供了两个示例配置文件：

- `client_config.json` - 客户端配置
- `server_config.json` - 服务器配置

### 运行

#### 启动服务器
```bash
make run-server
# 或者
./build/T2D -config server_config.json
```

#### 启动客户端
```bash
make run-client
# 或者
./build/T2D -config client_config.json
```

## 配置说明

### 基础配置

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `mode` | string | 是 | 运行模式："client" 或 "server" |
| `log_level` | string | 否 | 日志级别："debug", "info", "warn", "error" |
| `session_timeout` | int | 否 | UDP会话超时时间（秒），默认300 |
| `reconnect_interval` | int | 否 | TCP重连间隔（秒），默认5 |

### 客户端配置

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `listen_addr` | string | 是 | UDP监听地址，如 "0.0.0.0:51820" |
| `server_upstream` | string | 是 | 服务器上行地址，如 "192.168.1.100:9001" |
| `server_downstream` | string | 是 | 服务器下行地址，如 "192.168.1.100:9002" |

### 服务器配置

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `upstream_port` | string | 是 | 上行监听端口，如 ":9001" |
| `downstream_port` | string | 是 | 下行监听端口，如 ":9002" |
| `backend_addr` | string | 是 | 后端UDP地址，如 "127.0.0.1:51820" |

### 加密配置

#### 双向独立加密

T2D 的核心特性之一是支持上行和下行数据流使用完全独立的加密配置：

```json
{
  "upstream_crypto": {
    "type": "aes-256-gcm",
    "password": "upstream-secret-key-2024"
  },
  "downstream_crypto": {
    "type": "chacha20-poly1305",
    "password": "downstream-secret-key-2024"
  }
}
```

**重要说明**:
- **上行加密**: 客户端 → 服务器方向的数据加密
- **下行加密**: 服务器 → 客户端方向的数据加密
- **独立密钥**: 上行和下行可以使用完全不同的密码和算法
- **配置匹配**: 客户端和服务器的加密配置必须完全一致

#### 加密配置示例

**高安全性配置** (不同算法+不同密钥):
```json
{
  "upstream_crypto": {
    "type": "aes-256-gcm",
    "password": "AES-upstream-key-32chars-long!!!"
  },
  "downstream_crypto": {
    "type": "xchacha20-poly1305",
    "password": "XChaCha20-downstream-key-secure"
  }
}
```

**性能优化配置** (快速算法):
```json
{
  "upstream_crypto": {
    "type": "chacha20-poly1305",
    "password": "fast-upstream-encryption-key"
  },
  "downstream_crypto": {
    "type": "chacha20-poly1305",
    "password": "fast-downstream-encryption-key"
  }
}
```

**无加密配置** (明文传输):
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

#### 支持的加密算法

- `none` - 无加密
- `xor` - XOR混淆
- `aes-128-gcm` - AES-128-GCM
- `aes-256-gcm` - AES-256-GCM
- `chacha20-poly1305` - ChaCha20-Poly1305
- `xchacha20-poly1305` - XChaCha20-Poly1305

### 性能调优配置

```json
{
  "multicore": true,
  "load_balancing": "least_connections",
  "socket_recv_buffer": 65536,
  "socket_send_buffer": 65536,
  "tcp_keep_alive": 30,
  "tcp_no_delay": true
}
```

## 使用场景

### 1. WireGuard 代理

将 WireGuard UDP 流量通过 TCP 传输，适用于限制 UDP 的网络环境：

```bash
# 服务器端
./T2D -config server_config.json

# 客户端
./T2D -config client_config.json

# WireGuard 客户端连接到 T2D 客户端的监听地址
wg-quick up wg0  # 配置 endpoint 为 127.0.0.1:51820
```

### 2. 游戏加速

为 UDP 游戏流量提供 TCP 通道和加密：

```json
{
  "mode": "client",
  "listen_addr": "0.0.0.0:7777",
  "server_upstream": "game-server.com:9001",
  "server_downstream": "game-server.com:9002",
  "upstream_crypto": {
    "type": "chacha20-poly1305",
    "password": "game-secret-key"
  }
}
```

### 3. DNS over TCP

将 DNS UDP 查询转换为 TCP 传输：

```json
{
  "mode": "server",
  "upstream_port": ":5353",
  "downstream_port": ":5354",
  "backend_addr": "8.8.8.8:53"
}
```

## 开发

### 项目结构

```
T2D/
├── cmd/t2d/           # 主程序入口
├── internal/
│   ├── app/           # 应用程序逻辑
│   ├── client/        # 客户端实现
│   ├── server/        # 服务器实现
│   ├── config/        # 配置管理
│   ├── crypto/        # 加密模块
│   ├── protocol/      # 协议定义
│   ├── session/       # 会话管理
│   └── stats/         # 统计信息
├── client_config.json # 客户端配置示例
├── server_config.json # 服务器配置示例
├── CONFIG_GUIDE.md    # 详细配置指南
├── CRYPTO_README.md   # 加密功能说明
└── Makefile          # 构建脚本
```

### 可用的 Make 命令

```bash
make build          # 构建项目
make clean          # 清理构建文件
make run-client     # 运行客户端
make run-server     # 运行服务器
make fmt            # 格式化代码
make vet            # 代码检查
make test           # 运行测试
make install        # 安装到系统
```

### 依赖项

- [gnet/v2](https://github.com/panjf2000/gnet) - 高性能网络框架
- [golang.org/x/crypto](https://golang.org/x/crypto) - Go 加密库

## 性能特性

- **高并发**: 基于 gnet 的事件驱动架构
- **内存优化**: 零拷贝和对象池技术
- **负载均衡**: 支持多种负载均衡策略
- **连接复用**: TCP 连接复用和保活
- **缓冲优化**: 可配置的读写缓冲区

## 安全特性

- **端到端加密**: 支持多种现代加密算法
- **密钥管理**: 安全的密钥派生和管理
- **会话隔离**: 独立的会话管理和超时
- **防重放**: 序列号和时间戳验证

## 故障排除

### 常见问题

1. **连接失败**
   - 检查防火墙设置
   - 验证配置文件中的地址和端口
   - 确认服务器端已启动

2. **加密错误**
   - 确保客户端和服务器使用相同的加密配置
   - 检查密码是否匹配
   - 验证加密算法是否支持

3. **性能问题**
   - 调整缓冲区大小
   - 启用多核模式
   - 优化负载均衡策略

### 日志分析

启用详细日志：
```json
{
  "log_level": "debug",
  "log_path": "/var/log/t2d.log"
}
```

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

本项目采用 Apache License 2.0 许可证。详见 [LICENSE](LICENSE) 文件。

```
Copyright 2024 T2D-GNET Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

## 更多文档

- [详细配置指南](CONFIG_GUIDE.md)
- [加密功能说明](CRYPTO_README.md)