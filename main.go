package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/panjf2000/gnet/v2"
	"github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
)

// Config 配置文件结构
type Config struct {
	Mode              string `json:"mode"`               // "client" or "server"
	ListenAddr        string `json:"listen_addr"`        // 客户端UDP监听地址
	ServerUpstream    string `json:"server_upstream"`    // 服务器上行地址 (客户端用)
	ServerDownstream  string `json:"server_downstream"`  // 服务器下行地址 (客户端用)
	UpstreamPort      string `json:"upstream_port"`      // 服务端上行监听端口
	DownstreamPort    string `json:"downstream_port"`    // 服务端下行监听端口
	BackendAddr       string `json:"backend_addr"`       // 服务端后端UDP地址
	ReconnectInterval int    `json:"reconnect_interval"` // TCP重连间隔（秒）
	SessionTimeout    int    `json:"session_timeout"`    // UDP会话超时（秒）
	BufferSize        int    `json:"buffer_size"`        // 缓冲区大小
	MaxPacketSize     int    `json:"max_packet_size"`    // 最大包大小
	LogLevel          string `json:"log_level"`          // 日志级别
}

// PacketHeader TCP包头结构
type PacketHeader struct {
	Magic      uint32 // 魔术数，用于校验
	Version    uint8  // 协议版本
	Type       uint8  // 包类型
	Flags      uint16 // 标志位
	SessionID  uint32 // 会话ID
	SequenceID uint32 // 序列号
	Length     uint32 // 数据长度
}

const (
	MagicNumber     = 0x54435050 // "TCPP"
	ProtocolVersion = 1
	HeaderSize      = 20

	// 包类型
	PacketTypeData      = 1
	PacketTypeHeartbeat = 2
	PacketTypeControl   = 3
)

// Stats 统计信息
type Stats struct {
	BytesSent       atomic.Uint64
	BytesReceived   atomic.Uint64
	PacketsSent     atomic.Uint64
	PacketsReceived atomic.Uint64
	Errors          atomic.Uint64
	ActiveSessions  atomic.Int32
}

// UDPSession UDP会话
type UDPSession struct {
	mu         sync.RWMutex
	sessionID  uint32
	conn       *net.UDPConn
	remoteAddr *net.UDPAddr
	lastActive time.Time
	sequenceID uint32
}

// Client 客户端结构
type Client struct {
	config         *Config
	udpListener    *net.UDPConn
	sessions       sync.Map // map[string]*UDPSession
	sessionCounter uint32
	upstreamConn   net.Conn // 到服务器的上行连接
	downstreamConn net.Conn // 到服务器的下行连接
	upstreamMu     sync.Mutex
	downstreamMu   sync.Mutex
	ctx            context.Context
	cancel         context.CancelFunc
	stats          *Stats
	pool           *goroutine.Pool
}

// Server 服务端结构
type Server struct {
	config            *Config
	sessions          sync.Map // map[uint32]*BackendSession
	upstreamEngine    gnet.Engine
	downstreamEngine  gnet.Engine
	upstreamRunning   atomic.Bool
	downstreamRunning atomic.Bool
	connMap           sync.Map // map[gnet.Conn]uint32 连接到会话的映射
	ctx               context.Context
	cancel            context.CancelFunc
	stats             *Stats
	pool              *goroutine.Pool
}

// BackendSession 后端会话
type BackendSession struct {
	mu             sync.RWMutex
	sessionID      uint32
	conn           *net.UDPConn
	backendAddr    *net.UDPAddr
	lastActive     time.Time
	sequenceID     uint32
	downstreamConn gnet.Conn // 下行连接
}

// UpstreamHandler 服务端上行处理器
type UpstreamHandler struct {
	server *Server
	pool   *goroutine.Pool
}

// DownstreamHandler 服务端下行处理器
type DownstreamHandler struct {
	server *Server
	pool   *goroutine.Pool
}

// Marshal 序列化包头
func (h *PacketHeader) Marshal() []byte {
	buf := make([]byte, HeaderSize)
	binary.BigEndian.PutUint32(buf[0:4], h.Magic)
	buf[4] = h.Version
	buf[5] = h.Type
	binary.BigEndian.PutUint16(buf[6:8], h.Flags)
	binary.BigEndian.PutUint32(buf[8:12], h.SessionID)
	binary.BigEndian.PutUint32(buf[12:16], h.SequenceID)
	binary.BigEndian.PutUint32(buf[16:20], h.Length)
	return buf
}

// Unmarshal 反序列化包头
func (h *PacketHeader) Unmarshal(data []byte) error {
	if len(data) < HeaderSize {
		return fmt.Errorf("invalid header size")
	}
	h.Magic = binary.BigEndian.Uint32(data[0:4])
	h.Version = data[4]
	h.Type = data[5]
	h.Flags = binary.BigEndian.Uint16(data[6:8])
	h.SessionID = binary.BigEndian.Uint32(data[8:12])
	h.SequenceID = binary.BigEndian.Uint32(data[12:16])
	h.Length = binary.BigEndian.Uint32(data[16:20])
	return nil
}

// loadConfig 加载配置文件
func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// 设置默认值
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

	return &config, nil
}

// ====== 上行处理器实现 ======
func (h *UpstreamHandler) OnBoot(eng gnet.Engine) gnet.Action {
	h.server.upstreamEngine = eng
	h.server.upstreamRunning.Store(true)
	log.Printf("[INFO] Upstream server listening on %s", h.server.config.UpstreamPort)
	return gnet.None
}

func (h *UpstreamHandler) OnShutdown(eng gnet.Engine) {
	log.Printf("[INFO] Upstream server shutting down")
	h.server.upstreamRunning.Store(false)
}

func (h *UpstreamHandler) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	log.Printf("[INFO] New upstream connection from %s", c.RemoteAddr())
	c.SetContext(&bytes.Buffer{})
	return nil, gnet.None
}

func (h *UpstreamHandler) OnClose(c gnet.Conn, err error) gnet.Action {
	if err != nil {
		log.Printf("[ERROR] Upstream connection closed with error: %s, %v", c.RemoteAddr(), err)
	} else {
		log.Printf("[INFO] Upstream connection closed: %s", c.RemoteAddr())
	}
	return gnet.None
}

func (h *UpstreamHandler) OnTraffic(c gnet.Conn) gnet.Action {
	buf := c.Context().(*bytes.Buffer)
	data, _ := c.Next(-1)
	buf.Write(data)

	for buf.Len() >= HeaderSize {
		headerData := buf.Bytes()[:HeaderSize]
		header := &PacketHeader{}
		if err := header.Unmarshal(headerData); err != nil {
			log.Printf("[ERROR] Invalid packet header: %v", err)
			buf.Reset()
			return gnet.None
		}

		if header.Magic != MagicNumber {
			log.Printf("[ERROR] Invalid magic number: %x", header.Magic)
			buf.Reset()
			return gnet.None
		}

		totalSize := HeaderSize + int(header.Length)
		if buf.Len() < totalSize {
			break
		}

		buf.Next(HeaderSize)
		packetData := make([]byte, header.Length)
		buf.Read(packetData)

		// 异步处理数据包
		h.pool.Submit(func() {
			h.server.handleUpstreamPacket(c, header, packetData)
		})
	}

	return gnet.None
}

func (h *UpstreamHandler) OnTick() (delay time.Duration, action gnet.Action) {
	return time.Second, gnet.None
}

// ====== 下行处理器实现 ======
func (h *DownstreamHandler) OnBoot(eng gnet.Engine) gnet.Action {
	h.server.downstreamEngine = eng
	h.server.downstreamRunning.Store(true)
	log.Printf("[INFO] Downstream server listening on %s", h.server.config.DownstreamPort)
	return gnet.None
}

func (h *DownstreamHandler) OnShutdown(eng gnet.Engine) {
	log.Printf("[INFO] Downstream server shutting down")
	h.server.downstreamRunning.Store(false)
}

func (h *DownstreamHandler) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	log.Printf("[INFO] New downstream connection from %s", c.RemoteAddr())
	c.SetContext(&bytes.Buffer{})
	return nil, gnet.None
}

func (h *DownstreamHandler) OnClose(c gnet.Conn, err error) gnet.Action {
	// 清理相关的会话映射
	if sessionID, ok := h.server.connMap.Load(c); ok {
		h.server.connMap.Delete(c)
		if v, ok := h.server.sessions.Load(sessionID); ok {
			session := v.(*BackendSession)
			session.mu.Lock()
			if session.downstreamConn == c {
				session.downstreamConn = nil
			}
			session.mu.Unlock()
		}
	}

	if err != nil {
		log.Printf("[ERROR] Downstream connection closed with error: %s, %v", c.RemoteAddr(), err)
	} else {
		log.Printf("[INFO] Downstream connection closed: %s", c.RemoteAddr())
	}
	return gnet.None
}

func (h *DownstreamHandler) OnTraffic(c gnet.Conn) gnet.Action {
	// 下行连接主要用于发送数据，接收的可能是心跳或控制消息
	buf := c.Context().(*bytes.Buffer)
	data, _ := c.Next(-1)
	buf.Write(data)

	for buf.Len() >= HeaderSize {
		headerData := buf.Bytes()[:HeaderSize]
		header := &PacketHeader{}
		if err := header.Unmarshal(headerData); err != nil {
			buf.Reset()
			return gnet.None
		}

		if header.Magic != MagicNumber {
			buf.Reset()
			return gnet.None
		}

		totalSize := HeaderSize + int(header.Length)
		if buf.Len() < totalSize {
			break
		}

		buf.Next(HeaderSize)
		packetData := make([]byte, header.Length)
		buf.Read(packetData)

		// 处理心跳等控制消息
		if header.Type == PacketTypeHeartbeat {
			// 可以更新连接活跃时间等
			log.Printf("[DEBUG] Received heartbeat from downstream connection")
		}

		// 将下行连接与会话关联
		h.server.connMap.Store(c, header.SessionID)
		if v, ok := h.server.sessions.Load(header.SessionID); ok {
			session := v.(*BackendSession)
			session.mu.Lock()
			session.downstreamConn = c
			session.mu.Unlock()
		}
	}

	return gnet.None
}

func (h *DownstreamHandler) OnTick() (delay time.Duration, action gnet.Action) {
	return time.Second, gnet.None
}

// ====== 客户端实现 ======
func NewClient(config *Config) (*Client, error) {
	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		config: config,
		ctx:    ctx,
		cancel: cancel,
		stats:  &Stats{},
		pool:   goroutine.Default(),
	}

	// 创建UDP监听
	addr, err := net.ResolveUDPAddr("udp", config.ListenAddr)
	if err != nil {
		return nil, fmt.Errorf("resolve UDP address: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen UDP: %w", err)
	}

	conn.SetReadBuffer(config.BufferSize)
	conn.SetWriteBuffer(config.BufferSize)
	client.udpListener = conn

	return client, nil
}

func (c *Client) Run() error {
	log.Printf("[INFO] Client starting, listening on %s", c.config.ListenAddr)

	// 启动TCP连接管理
	go c.manageTCPConnections()

	// 启动UDP接收
	go c.handleUDP()

	// 启动统计输出
	go c.printStats()

	// 启动会话清理
	go c.cleanupSessions()

	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("[INFO] Shutting down client...")
	c.cancel()
	c.udpListener.Close()
	if c.upstreamConn != nil {
		c.upstreamConn.Close()
	}
	if c.downstreamConn != nil {
		c.downstreamConn.Close()
	}

	return nil
}

func (c *Client) manageTCPConnections() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		// 连接到服务器的上行端口
		if c.upstreamConn == nil {
			log.Printf("[INFO] Connecting to server upstream %s", c.config.ServerUpstream)
			conn, err := net.DialTimeout("tcp", c.config.ServerUpstream, 10*time.Second)
			if err != nil {
				log.Printf("[ERROR] Failed to connect upstream: %v", err)
			} else {
				c.upstreamMu.Lock()
				c.upstreamConn = conn
				c.upstreamMu.Unlock()
				log.Printf("[INFO] Connected to server upstream %s", c.config.ServerUpstream)
			}
		}

		// 连接到服务器的下行端口
		if c.downstreamConn == nil {
			log.Printf("[INFO] Connecting to server downstream %s", c.config.ServerDownstream)
			conn, err := net.DialTimeout("tcp", c.config.ServerDownstream, 10*time.Second)
			if err != nil {
				log.Printf("[ERROR] Failed to connect downstream: %v", err)
			} else {
				c.downstreamMu.Lock()
				c.downstreamConn = conn
				c.downstreamMu.Unlock()
				log.Printf("[INFO] Connected to server downstream %s", c.config.ServerDownstream)
				go c.handleDownstream(conn)
			}
		}

		time.Sleep(time.Duration(c.config.ReconnectInterval) * time.Second)
	}
}

func (c *Client) handleUDP() {
	buf := make([]byte, c.config.MaxPacketSize)
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		n, addr, err := c.udpListener.ReadFromUDP(buf)
		if err != nil {
			if !isClosedError(err) {
				log.Printf("[ERROR] UDP read error: %v", err)
				c.stats.Errors.Add(1)
			}
			continue
		}

		c.stats.BytesReceived.Add(uint64(n))
		c.stats.PacketsReceived.Add(1)

		// 获取或创建会话
		sessionKey := addr.String()
		var session *UDPSession
		if v, ok := c.sessions.Load(sessionKey); ok {
			session = v.(*UDPSession)
			session.mu.Lock()
			session.lastActive = time.Now()
			session.mu.Unlock()
		} else {
			sessionID := atomic.AddUint32(&c.sessionCounter, 1)
			session = &UDPSession{
				sessionID:  sessionID,
				conn:       c.udpListener,
				remoteAddr: addr,
				lastActive: time.Now(),
			}
			c.sessions.Store(sessionKey, session)
			c.stats.ActiveSessions.Add(1)
			log.Printf("[INFO] New UDP session %d from %s", sessionID, addr)
		}

		data := make([]byte, n)
		copy(data, buf[:n])

		// 异步发送到上行
		c.pool.Submit(func() {
			c.sendToUpstream(session, data)
		})
	}
}

func (c *Client) sendToUpstream(session *UDPSession, data []byte) {
	c.upstreamMu.Lock()
	defer c.upstreamMu.Unlock()

	if c.upstreamConn == nil {
		log.Printf("[WARN] Upstream connection not available")
		c.stats.Errors.Add(1)
		return
	}

	session.mu.Lock()
	sequenceID := session.sequenceID
	session.sequenceID++
	session.mu.Unlock()

	header := &PacketHeader{
		Magic:      MagicNumber,
		Version:    ProtocolVersion,
		Type:       PacketTypeData,
		Flags:      0,
		SessionID:  session.sessionID,
		SequenceID: sequenceID,
		Length:     uint32(len(data)),
	}

	packet := append(header.Marshal(), data...)
	if _, err := c.upstreamConn.Write(packet); err != nil {
		log.Printf("[ERROR] Upstream write error: %v", err)
		c.stats.Errors.Add(1)
		c.upstreamConn.Close()
		c.upstreamConn = nil
		return
	}

	c.stats.BytesSent.Add(uint64(len(packet)))
	c.stats.PacketsSent.Add(1)
}

func (c *Client) handleDownstream(conn net.Conn) {
	defer func() {
		conn.Close()
		c.downstreamMu.Lock()
		if c.downstreamConn == conn {
			c.downstreamConn = nil
		}
		c.downstreamMu.Unlock()
	}()

	// 首先发送一个心跳包，让服务器知道这是哪个客户端
	c.sendHeartbeat(conn)

	buf := make([]byte, c.config.BufferSize)
	readBuf := bytes.NewBuffer(nil)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		n, err := conn.Read(buf)
		if err != nil {
			if !isTimeoutError(err) && !isClosedError(err) {
				log.Printf("[ERROR] Downstream read error: %v", err)
				c.stats.Errors.Add(1)
			}
			return
		}

		readBuf.Write(buf[:n])
		c.stats.BytesReceived.Add(uint64(n))

		// 解析包
		for readBuf.Len() >= HeaderSize {
			headerData := readBuf.Bytes()[:HeaderSize]
			header := &PacketHeader{}
			if err := header.Unmarshal(headerData); err != nil {
				log.Printf("[ERROR] Invalid downstream packet header: %v", err)
				readBuf.Reset()
				break
			}

			if header.Magic != MagicNumber {
				log.Printf("[ERROR] Invalid downstream magic number: %x", header.Magic)
				readBuf.Reset()
				break
			}

			totalSize := HeaderSize + int(header.Length)
			if readBuf.Len() < totalSize {
				break
			}

			readBuf.Next(HeaderSize)
			data := make([]byte, header.Length)
			readBuf.Read(data)

			c.stats.PacketsReceived.Add(1)

			if header.Type == PacketTypeData {
				c.pool.Submit(func() {
					c.handleDownstreamPacket(header, data)
				})
			}
		}
	}
}

func (c *Client) sendHeartbeat(conn net.Conn) {
	// 发送一个心跳包，包含一个活跃的会话ID
	var sessionID uint32
	c.sessions.Range(func(key, value interface{}) bool {
		session := value.(*UDPSession)
		sessionID = session.sessionID
		return false
	})

	header := &PacketHeader{
		Magic:      MagicNumber,
		Version:    ProtocolVersion,
		Type:       PacketTypeHeartbeat,
		Flags:      0,
		SessionID:  sessionID,
		SequenceID: 0,
		Length:     0,
	}

	conn.Write(header.Marshal())
}

func (c *Client) handleDownstreamPacket(header *PacketHeader, data []byte) {
	// 查找会话
	var targetSession *UDPSession
	c.sessions.Range(func(key, value interface{}) bool {
		session := value.(*UDPSession)
		if session.sessionID == header.SessionID {
			targetSession = session
			return false
		}
		return true
	})

	if targetSession == nil {
		log.Printf("[WARN] Session %d not found for downstream packet", header.SessionID)
		c.stats.Errors.Add(1)
		return
	}

	// 发送到UDP客户端
	if _, err := targetSession.conn.WriteToUDP(data, targetSession.remoteAddr); err != nil {
		log.Printf("[ERROR] UDP write error: %v", err)
		c.stats.Errors.Add(1)
		return
	}

	c.stats.BytesSent.Add(uint64(len(data)))
	c.stats.PacketsSent.Add(1)

	targetSession.mu.Lock()
	targetSession.lastActive = time.Now()
	targetSession.mu.Unlock()
}

func (c *Client) cleanupSessions() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			timeout := time.Duration(c.config.SessionTimeout) * time.Second

			var toDelete []string
			c.sessions.Range(func(key, value interface{}) bool {
				session := value.(*UDPSession)
				session.mu.RLock()
				lastActive := session.lastActive
				session.mu.RUnlock()

				if now.Sub(lastActive) > timeout {
					toDelete = append(toDelete, key.(string))
					log.Printf("[INFO] Session %d timeout", session.sessionID)
				}
				return true
			})

			for _, key := range toDelete {
				c.sessions.Delete(key)
				c.stats.ActiveSessions.Add(-1)
			}
		}
	}
}

func (c *Client) printStats() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			log.Printf("[STATS] Client - Sessions: %d, Sent: %d bytes/%d packets, Received: %d bytes/%d packets, Errors: %d",
				c.stats.ActiveSessions.Load(),
				c.stats.BytesSent.Load(),
				c.stats.PacketsSent.Load(),
				c.stats.BytesReceived.Load(),
				c.stats.PacketsReceived.Load(),
				c.stats.Errors.Load())
		}
	}
}

// ====== 服务端实现 ======
func NewServer(config *Config) (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())

	server := &Server{
		config: config,
		ctx:    ctx,
		cancel: cancel,
		stats:  &Stats{},
		pool:   goroutine.Default(),
	}

	return server, nil
}

func (s *Server) Run() error {
	log.Printf("[INFO] Server starting")

	// 启动上行TCP服务器
	go s.startUpstreamServer()

	// 启动下行TCP服务器
	go s.startDownstreamServer()

	// 启动统计输出
	go s.printStats()

	// 启动会话清理
	go s.cleanupSessions()

	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("[INFO] Shutting down server...")
	s.cancel()
	if s.upstreamRunning.Load() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.upstreamEngine.Stop(ctx)
	}
	if s.downstreamRunning.Load() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.downstreamEngine.Stop(ctx)
	}

	return nil
}

func (s *Server) startUpstreamServer() {
	handler := &UpstreamHandler{
		server: s,
		pool:   s.pool,
	}

	options := []gnet.Option{
		gnet.WithMulticore(true),
		gnet.WithReusePort(true),
		gnet.WithTCPKeepAlive(30 * time.Second),
		gnet.WithReadBufferCap(s.config.BufferSize),
	}

	err := gnet.Run(handler, "tcp://"+s.config.UpstreamPort, options...)
	if err != nil {
		log.Fatalf("[FATAL] Upstream server error: %v", err)
	}
}

func (s *Server) startDownstreamServer() {
	handler := &DownstreamHandler{
		server: s,
		pool:   s.pool,
	}

	options := []gnet.Option{
		gnet.WithMulticore(true),
		gnet.WithReusePort(true),
		gnet.WithTCPKeepAlive(30 * time.Second),
		gnet.WithReadBufferCap(s.config.BufferSize),
	}

	err := gnet.Run(handler, "tcp://"+s.config.DownstreamPort, options...)
	if err != nil {
		log.Fatalf("[FATAL] Downstream server error: %v", err)
	}
}

func (s *Server) handleUpstreamPacket(conn gnet.Conn, header *PacketHeader, data []byte) {
	s.stats.PacketsReceived.Add(1)
	s.stats.BytesReceived.Add(uint64(HeaderSize + len(data)))

	// 查找或创建后端会话
	var session *BackendSession
	if v, ok := s.sessions.Load(header.SessionID); ok {
		session = v.(*BackendSession)
		session.mu.Lock()
		session.lastActive = time.Now()
		session.mu.Unlock()
	} else {
		// 创建新会话
		backendAddr, err := net.ResolveUDPAddr("udp", s.config.BackendAddr)
		if err != nil {
			log.Printf("[ERROR] Failed to resolve backend address: %v", err)
			s.stats.Errors.Add(1)
			return
		}

		udpConn, err := net.DialUDP("udp", nil, backendAddr)
		if err != nil {
			log.Printf("[ERROR] Failed to dial backend: %v", err)
			s.stats.Errors.Add(1)
			return
		}

		udpConn.SetReadBuffer(s.config.BufferSize)
		udpConn.SetWriteBuffer(s.config.BufferSize)

		session = &BackendSession{
			sessionID:   header.SessionID,
			conn:        udpConn,
			backendAddr: backendAddr,
			lastActive:  time.Now(),
		}
		s.sessions.Store(header.SessionID, session)
		s.stats.ActiveSessions.Add(1)

		// 启动后端数据接收
		go s.handleBackendData(session)

		log.Printf("[INFO] New backend session %d to %s", header.SessionID, backendAddr)
	}

	// 发送到后端
	if _, err := session.conn.Write(data); err != nil {
		log.Printf("[ERROR] Backend write error: %v", err)
		s.stats.Errors.Add(1)
		return
	}

	s.stats.BytesSent.Add(uint64(len(data)))
	s.stats.PacketsSent.Add(1)
}

func (s *Server) handleBackendData(session *BackendSession) {
	defer func() {
		session.conn.Close()
		s.sessions.Delete(session.sessionID)
		s.stats.ActiveSessions.Add(-1)
		log.Printf("[INFO] Backend session %d closed", session.sessionID)
	}()

	buf := make([]byte, s.config.MaxPacketSize)
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		session.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		n, err := session.conn.Read(buf)
		if err != nil {
			if !isTimeoutError(err) && !isClosedError(err) {
				log.Printf("[ERROR] Backend read error: %v", err)
				s.stats.Errors.Add(1)
			}
			return
		}

		s.stats.BytesReceived.Add(uint64(n))
		s.stats.PacketsReceived.Add(1)

		// 复制数据
		data := make([]byte, n)
		copy(data, buf[:n])

		// 异步发送到下行
		s.pool.Submit(func() {
			s.sendToDownstream(session, data)
		})

		// 更新活跃时间
		session.mu.Lock()
		session.lastActive = time.Now()
		session.mu.Unlock()
	}
}

func (s *Server) sendToDownstream(session *BackendSession, data []byte) {
	session.mu.RLock()
	downConn := session.downstreamConn
	sessionID := session.sessionID
	session.mu.RUnlock()

	if downConn == nil {
		// 尝试从连接映射中查找
		var found bool
		s.connMap.Range(func(key, value interface{}) bool {
			if value.(uint32) == sessionID {
				downConn = key.(gnet.Conn)
				found = true
				return false
			}
			return true
		})

		if !found {
			log.Printf("[WARN] No downstream connection for session %d", sessionID)
			s.stats.Errors.Add(1)
			return
		}
	}

	// 构建包头
	session.mu.Lock()
	sequenceID := session.sequenceID
	session.sequenceID++
	session.mu.Unlock()

	header := &PacketHeader{
		Magic:      MagicNumber,
		Version:    ProtocolVersion,
		Type:       PacketTypeData,
		Flags:      0,
		SessionID:  sessionID,
		SequenceID: sequenceID,
		Length:     uint32(len(data)),
	}

	// 发送数据
	packet := append(header.Marshal(), data...)
	if err := downConn.AsyncWrite(packet, nil); err != nil {
		log.Printf("[ERROR] Downstream write error: %v", err)
		s.stats.Errors.Add(1)
		return
	}

	s.stats.BytesSent.Add(uint64(len(packet)))
	s.stats.PacketsSent.Add(1)
}

func (s *Server) cleanupSessions() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			timeout := time.Duration(s.config.SessionTimeout) * time.Second

			var toDelete []uint32
			s.sessions.Range(func(key, value interface{}) bool {
				session := value.(*BackendSession)
				session.mu.RLock()
				lastActive := session.lastActive
				session.mu.RUnlock()

				if now.Sub(lastActive) > timeout {
					toDelete = append(toDelete, key.(uint32))
					log.Printf("[INFO] Backend session %d timeout", session.sessionID)
				}
				return true
			})

			for _, sessionID := range toDelete {
				if v, ok := s.sessions.Load(sessionID); ok {
					session := v.(*BackendSession)
					session.conn.Close()
					s.sessions.Delete(sessionID)
					s.stats.ActiveSessions.Add(-1)
				}
			}
		}
	}
}

func (s *Server) printStats() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			log.Printf("[STATS] Server - Sessions: %d, Sent: %d bytes/%d packets, Received: %d bytes/%d packets, Errors: %d",
				s.stats.ActiveSessions.Load(),
				s.stats.BytesSent.Load(),
				s.stats.PacketsSent.Load(),
				s.stats.BytesReceived.Load(),
				s.stats.PacketsReceived.Load(),
				s.stats.Errors.Load())
		}
	}
}

// 辅助函数
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	netErr, ok := err.(net.Error)
	return ok && netErr.Timeout()
}

func isClosedError(err error) bool {
	if err == nil {
		return false
	}
	return err == net.ErrClosed || err.Error() == "use of closed network connection"
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: forwarder <config.json>")
		os.Exit(1)
	}

	// 加载配置
	config, err := loadConfig(os.Args[1])
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// 设置日志级别
	switch config.LogLevel {
	case "debug":
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	case "info":
		log.SetFlags(log.LstdFlags)
	case "warn", "error":
		log.SetFlags(log.LstdFlags)
	}

	// 根据模式启动
	switch config.Mode {
	case "client":
		client, err := NewClient(config)
		if err != nil {
			log.Fatal("Failed to create client:", err)
		}
		if err := client.Run(); err != nil {
			log.Fatal("Client error:", err)
		}
	case "server":
		server, err := NewServer(config)
		if err != nil {
			log.Fatal("Failed to create server:", err)
		}
		if err := server.Run(); err != nil {
			log.Fatal("Server error:", err)
		}
	default:
		log.Fatal("Invalid mode:", config.Mode)
	}
}
