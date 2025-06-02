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

// Config 配置文件结构
type Config struct {
	Mode              string       `json:"mode"`               // "client" or "server"
	ListenAddr        string       `json:"listen_addr"`        // 客户端UDP监听地址
	ServerUpstream    string       `json:"server_upstream"`    // 服务器上行地址 (客户端用)
	ServerDownstream  string       `json:"server_downstream"`  // 服务器下行地址 (客户端用)
	UpstreamPort      string       `json:"upstream_port"`      // 服务端上行监听端口
	DownstreamPort    string       `json:"downstream_port"`    // 服务端下行监听端口
	BackendAddr       string       `json:"backend_addr"`       // 服务端后端UDP地址
	ReconnectInterval int          `json:"reconnect_interval"` // TCP重连间隔（秒）
	SessionTimeout    int          `json:"session_timeout"`    // UDP会话超时（秒）
	BufferSize        int          `json:"buffer_size"`        // 缓冲区大小
	MaxPacketSize     int          `json:"max_packet_size"`    // 最大包大小
	LogLevel          string       `json:"log_level"`          // 日志级别
	UpstreamCrypto    CryptoConfig `json:"upstream_crypto"`    // 上行加密配置
	DownstreamCrypto  CryptoConfig `json:"downstream_crypto"`  // 下行加密配置

	// gnet相关配置
	Multicore            bool   `json:"multicore"`
	LockOSThread         bool   `json:"lock_os_thread"`
	LoadBalancing        string `json:"load_balancing"`
	NumEventLoop         int    `json:"num_event_loop"`
	ReuseAddr            bool   `json:"reuse_addr"`
	ReusePort            bool   `json:"reuse_port"`
	SocketRecvBuffer     int    `json:"socket_recv_buffer"`
	SocketSendBuffer     int    `json:"socket_send_buffer"`
	TCPKeepAlive         int    `json:"tcp_keep_alive"`
	TCPNoDelay           bool   `json:"tcp_no_delay"`
	Ticker               bool   `json:"ticker"`
	ReadBufferCap        int    `json:"read_buffer_cap"`
	WriteBufferCap       int    `json:"write_buffer_cap"`
	EdgeTriggeredIO      bool   `json:"edge_triggered_io"`
	EdgeTriggeredIOChunk int    `json:"edge_triggered_io_chunk"`
	BindToDevice         string `json:"bind_to_device"`
	LogPath              string `json:"log_path"`
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
	if config.ReconnectInterval <= 0 {
		config.ReconnectInterval = 5
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
	if config.LoadBalancing == "" {
		config.LoadBalancing = "least_connections"
	}
	if config.SocketRecvBuffer <= 0 {
		config.SocketRecvBuffer = 65536
	}
	if config.SocketSendBuffer <= 0 {
		config.SocketSendBuffer = 65536
	}
	if config.TCPKeepAlive <= 0 {
		config.TCPKeepAlive = 30
	}
	if config.ReadBufferCap <= 0 {
		config.ReadBufferCap = 65536
	}
	if config.WriteBufferCap <= 0 {
		config.WriteBufferCap = 65536
	}
	if config.EdgeTriggeredIOChunk <= 0 {
		config.EdgeTriggeredIOChunk = 8192
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
