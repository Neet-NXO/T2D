package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// CryptoConfig 加密配置
type CryptoConfig struct {
	Type     string `json:"type"`               // 加密类型
	Password string `json:"password,omitempty"` // 密码（可选）
}

const (
	DefaultSocketBuffer        = 16 * 1024 * 1024
	DefaultTCPBuffer           = 16 * 1024 * 1024
	DefaultTCPKeepAliveSeconds = 30
	DefaultGnetReadBufferCap   = 1024 * 1024
	DefaultGnetWriteBufferCap  = 1024 * 1024
)

// Config 配置文件结构
type Config struct {
	Mode                      string       `json:"mode"`                   // "client" or "server"
	ListenAddr                string       `json:"listen_addr"`            // 客户端UDP监听地址
	ServerUpstream            string       `json:"server_upstream"`        // 服务器上行地址 (客户端用)
	ServerDownstream          string       `json:"server_downstream"`      // 服务器下行地址 (客户端用)
	UpstreamPort              string       `json:"upstream_port"`          // 服务端上行监听端口
	DownstreamPort            string       `json:"downstream_port"`        // 服务端下行监听端口
	BackendAddr               string       `json:"backend_addr"`           // 服务端后端UDP地址
	ReconnectInterval         int          `json:"reconnect_interval"`     // TCP重连初始间隔（秒）
	ReconnectMaxInterval      int          `json:"reconnect_max_interval"` // TCP最大重连间隔（秒）
	ReconnectJitterMs         int          `json:"reconnect_jitter_ms"`    // TCP重连抖动（毫秒）
	SessionTimeout            int          `json:"session_timeout"`        // UDP会话超时（秒）
	BufferSize                int          `json:"buffer_size"`            // 缓冲区大小
	MaxPacketSize             int          `json:"max_packet_size"`        // 最大包大小
	UpstreamQueueSize         int          `json:"-"`                      // 固定内置，不允许用户配置
	UpstreamBatchMaxPackets   int          `json:"-"`                      // 固定内置，不允许用户配置
	UpstreamBatchMaxBytes     int          `json:"-"`                      // 固定内置，不允许用户配置
	UpstreamBatchMaxDelayMs   int          `json:"-"`                      // 固定内置，不允许用户配置
	UpstreamStripeEnabled     bool         `json:"-"`                      // 固定内置，不允许用户配置
	UpstreamReorderWindow     int          `json:"-"`                      // 固定内置，不允许用户配置
	UpstreamReorderHoldMs     int          `json:"-"`                      // 固定内置，不允许用户配置
	UpstreamLaneCount         int          `json:"-"`                      // 固定内置，不允许用户配置
	DownstreamLaneCount       int          `json:"-"`                      // 固定内置，不允许用户配置
	DownstreamBatchMaxPackets int          `json:"-"`                      // 固定内置，不允许用户配置
	DownstreamBatchMaxBytes   int          `json:"-"`                      // 固定内置，不允许用户配置
	DownstreamBatchMaxDelayMs int          `json:"-"`                      // 固定内置，不允许用户配置
	HeartbeatIntervalMs       int          `json:"-"`                      // 固定内置，不允许用户配置
	LogLevel                  string       `json:"log_level"`              // 日志级别
	UpstreamCrypto            CryptoConfig `json:"upstream_crypto"`        // 上行加密配置
	DownstreamCrypto          CryptoConfig `json:"downstream_crypto"`      // 下行加密配置
}

// Load 加载配置文件
func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// 设置默认值
	setDefaults(&config)

	return &config, nil
}

// setDefaults 设置默认值
func setDefaults(config *Config) {
	// 性能关键参数全部固定内置，避免用户侧误配置导致吞吐/稳定性波动。
	config.UpstreamQueueSize = 8192
	config.UpstreamBatchMaxPackets = 256
	config.UpstreamBatchMaxBytes = 4 * 1024 * 1024
	config.UpstreamBatchMaxDelayMs = 0
	config.UpstreamStripeEnabled = false
	config.UpstreamReorderWindow = 128
	config.UpstreamReorderHoldMs = 3
	config.UpstreamLaneCount = 4
	config.DownstreamLaneCount = 4
	config.DownstreamBatchMaxPackets = 128
	config.DownstreamBatchMaxBytes = 512 * 1024
	config.DownstreamBatchMaxDelayMs = 1
	config.HeartbeatIntervalMs = 1000

	if config.ReconnectInterval <= 0 {
		config.ReconnectInterval = 1
	}
	if config.ReconnectMaxInterval <= 0 {
		config.ReconnectMaxInterval = 30
	}
	if config.ReconnectMaxInterval < config.ReconnectInterval {
		config.ReconnectMaxInterval = config.ReconnectInterval
	}
	if config.ReconnectJitterMs < 0 {
		config.ReconnectJitterMs = 0
	}
	if config.SessionTimeout <= 0 {
		config.SessionTimeout = 300
	}
	if config.BufferSize <= 0 {
		config.BufferSize = 65536
	}
	if config.MaxPacketSize <= 0 {
		config.MaxPacketSize = 65536
	}
	if config.LogLevel == "" {
		config.LogLevel = "info"
	}
	// 设置加密默认值
	if config.UpstreamCrypto.Type == "" {
		config.UpstreamCrypto.Type = "none"
	}
	if config.DownstreamCrypto.Type == "" {
		config.DownstreamCrypto.Type = "none"
	}
}

// IsClient 判断是否为客户端模式
func (c *Config) IsClient() bool {
	return c.Mode == "client"
}

// IsServer 判断是否为服务端模式
func (c *Config) IsServer() bool {
	return c.Mode == "server"
}
