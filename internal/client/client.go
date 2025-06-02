package client

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"t2d/internal/config"
	"t2d/internal/crypto"
	"t2d/internal/protocol"
	"t2d/internal/session"
	"t2d/internal/stats"

	"github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
)

// Client 客户端结构
type Client struct {
	config           *config.Config
	udpListener      *net.UDPConn
	sessions         sync.Map // map[string]*session.UDPSession
	sessionCounter   uint32
	upstreamConn     net.Conn // 到服务器的上行连接
	downstreamConn   net.Conn // 到服务器的下行连接
	upstreamMu       sync.Mutex
	downstreamMu     sync.Mutex
	ctx              context.Context
	cancel           context.CancelFunc
	stats            *stats.Stats
	pool             *goroutine.Pool
	upstreamCipher   crypto.Cipher // 上行加密器
	downstreamCipher crypto.Cipher // 下行加密器
}

// New 创建新的客户端实例
func New(cfg *config.Config) (*Client, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建协程池
	pool := goroutine.Default()

	// 创建上行加密器
	upstreamCryptoConfig := &crypto.Config{
		Type:     crypto.CipherType(cfg.UpstreamCrypto.Type),
		Password: cfg.UpstreamCrypto.Password,
	}
	upstreamCipher, err := crypto.NewCipher(upstreamCryptoConfig)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create upstream cipher: %w", err)
	}

	// 创建下行加密器
	downstreamCryptoConfig := &crypto.Config{
		Type:     crypto.CipherType(cfg.DownstreamCrypto.Type),
		Password: cfg.DownstreamCrypto.Password,
	}
	downstreamCipher, err := crypto.NewCipher(downstreamCryptoConfig)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create downstream cipher: %w", err)
	}

	return &Client{
		config:           cfg,
		ctx:              ctx,
		cancel:           cancel,
		stats:            stats.New(),
		pool:             pool,
		upstreamCipher:   upstreamCipher,
		downstreamCipher: downstreamCipher,
	}, nil
}

// Start 启动客户端
func (c *Client) Start() error {
	// 启动UDP监听
	udpAddr, err := net.ResolveUDPAddr("udp", c.config.ListenAddr)
	if err != nil {
		return err
	}

	c.udpListener, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Client listening on UDP %s", c.config.ListenAddr)

	// 启动TCP连接管理
	go c.manageTCPConnections()

	// 启动UDP处理
	go c.handleUDP()

	// 启动会话清理
	go c.cleanupSessions()

	// 启动统计输出
	go c.printStats()

	return nil
}

// Stop 停止客户端
func (c *Client) Stop() error {
	log.Println("[INFO] Shutting down client...")
	c.cancel()

	if c.udpListener != nil {
		c.udpListener.Close()
	}
	if c.upstreamConn != nil {
		c.upstreamConn.Close()
	}
	if c.downstreamConn != nil {
		c.downstreamConn.Close()
	}

	return nil
}

// Run 运行客户端（阻塞直到收到信号）
func (c *Client) Run() error {
	if err := c.Start(); err != nil {
		return err
	}

	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	return c.Stop()
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
				c.stats.AddError()
			}
			continue
		}

		c.stats.AddBytesReceived(uint64(n))
		c.stats.AddPacketsReceived(1)

		// 获取或创建会话
		sessionKey := addr.String()
		var sess *session.UDPSession
		if v, ok := c.sessions.Load(sessionKey); ok {
			sess = v.(*session.UDPSession)
			sess.UpdateLastActive()
		} else {
			sessionID := atomic.AddUint32(&c.sessionCounter, 1)
			sess = session.NewUDPSession(sessionID, c.udpListener, addr)
			c.sessions.Store(sessionKey, sess)
			c.stats.AddActiveSession()
			log.Printf("[INFO] New UDP session %d from %s", sessionID, addr)
		}

		data := make([]byte, n)
		copy(data, buf[:n])

		// 异步发送到上行
		c.pool.Submit(func() {
			c.sendToUpstream(sess, data)
		})
	}
}

func (c *Client) sendToUpstream(sess *session.UDPSession, data []byte) {
	c.upstreamMu.Lock()
	defer c.upstreamMu.Unlock()

	if c.upstreamConn == nil {
		log.Printf("[WARN] Upstream connection not available")
		c.stats.AddError()
		return
	}

	// 加密数据
	encryptedData, err := c.upstreamCipher.Encrypt(data)
	if err != nil {
		log.Printf("[ERROR] Failed to encrypt upstream data: %v", err)
		c.stats.AddError()
		return
	}

	sequenceID := sess.NextSequenceID()
	header := protocol.NewDataPacket(sess.GetSessionID(), sequenceID, uint32(len(encryptedData)))

	packet := append(header.Marshal(), encryptedData...)
	if _, err := c.upstreamConn.Write(packet); err != nil {
		log.Printf("[ERROR] Upstream write error: %v", err)
		c.stats.AddError()
		c.upstreamConn.Close()
		c.upstreamConn = nil
		return
	}

	c.stats.AddBytesSent(uint64(len(packet)))
	c.stats.AddPacketsSent(1)
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

	// 启动心跳发送协程
	heartbeatTicker := time.NewTicker(1 * time.Second)
	defer heartbeatTicker.Stop()
	go func() {
		for {
			select {
			case <-c.ctx.Done():
				return
			case <-heartbeatTicker.C:
				c.sendHeartbeat(conn)
			}
		}
	}()

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
				c.stats.AddError()
			}
			return
		}

		readBuf.Write(buf[:n])
		c.stats.AddBytesReceived(uint64(n))

		// 解析包
		for readBuf.Len() >= protocol.HeaderSize {
			headerData := readBuf.Bytes()[:protocol.HeaderSize]
			header := &protocol.PacketHeader{}
			if err := header.Unmarshal(headerData); err != nil {
				log.Printf("[ERROR] Invalid downstream packet header: %v", err)
				readBuf.Reset()
				break
			}

			if err := header.Validate(); err != nil {
				log.Printf("[ERROR] Invalid downstream packet: %v", err)
				readBuf.Reset()
				break
			}

			totalLen := protocol.HeaderSize + int(header.Length)
			if readBuf.Len() < totalLen {
				break // 等待更多数据
			}

			// 处理完整的包
			packetData := make([]byte, totalLen)
			readBuf.Read(packetData)

			c.pool.Submit(func() {
				c.handleDownstreamPacket(header, packetData[protocol.HeaderSize:])
			})
		}
	}
}

func (c *Client) handleDownstreamPacket(header *protocol.PacketHeader, data []byte) {
	if header.Type == protocol.PacketTypeHeartbeat {
		return // 忽略心跳包
	}

	// 解密数据
	decryptedData, err := c.downstreamCipher.Decrypt(data)
	if err != nil {
		log.Printf("[ERROR] Failed to decrypt downstream data: %v", err)
		c.stats.AddError()
		return
	}

	// 查找对应的UDP会话
	var targetSession *session.UDPSession
	c.sessions.Range(func(key, value interface{}) bool {
		sess := value.(*session.UDPSession)
		if sess.GetSessionID() == header.SessionID {
			targetSession = sess
			return false
		}
		return true
	})

	if targetSession == nil {
		log.Printf("[WARN] Session %d not found for downstream packet", header.SessionID)
		return
	}

	// 发送到UDP客户端
	if _, err := c.udpListener.WriteToUDP(decryptedData, targetSession.GetRemoteAddr()); err != nil {
		log.Printf("[ERROR] UDP write error: %v", err)
		c.stats.AddError()
		return
	}

	c.stats.AddBytesSent(uint64(len(decryptedData)))
	c.stats.AddPacketsSent(1)
	targetSession.UpdateLastActive()
}

func (c *Client) sendHeartbeat(conn net.Conn) {
	// 发送心跳包，包含所有活跃会话的ID
	c.sessions.Range(func(key, value interface{}) bool {
		sess := value.(*session.UDPSession)
		header := protocol.NewHeartbeatPacket(sess.GetSessionID())
		packet := header.Marshal()
		if _, err := conn.Write(packet); err != nil {
			log.Printf("[ERROR] Failed to send heartbeat for session %d: %v", sess.GetSessionID(), err)
		}
		return true
	})

	// 如果没有活跃会话，发送一个会话ID为0的心跳包
	var hasActiveSessions bool
	c.sessions.Range(func(key, value interface{}) bool {
		hasActiveSessions = true
		return false
	})

	if !hasActiveSessions {
		header := protocol.NewHeartbeatPacket(0)
		packet := header.Marshal()
		if _, err := conn.Write(packet); err != nil {
			log.Printf("[ERROR] Failed to send default heartbeat: %v", err)
		}
	}
}

func (c *Client) cleanupSessions() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			timeout := time.Duration(c.config.SessionTimeout) * time.Second
			var expiredKeys []interface{}

			c.sessions.Range(func(key, value interface{}) bool {
				sess := value.(*session.UDPSession)
				if sess.IsExpired(timeout) {
					expiredKeys = append(expiredKeys, key)
				}
				return true
			})

			for _, key := range expiredKeys {
				if v, ok := c.sessions.LoadAndDelete(key); ok {
					sess := v.(*session.UDPSession)
					log.Printf("[INFO] Cleaning up expired session %d", sess.GetSessionID())
					c.stats.RemoveActiveSession()
				}
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
			summary := c.stats.GetSummary()
			log.Printf("[STATS] Sessions: %d, Sent: %d bytes/%d packets, Received: %d bytes/%d packets, Errors: %d, Uptime: %v",
				summary.ActiveSessions,
				summary.BytesSent, summary.PacketsSent,
				summary.BytesReceived, summary.PacketsReceived,
				summary.Errors, summary.Uptime)
		}
	}
}

// 辅助函数
func isClosedError(err error) bool {
	return err != nil && (err.Error() == "use of closed network connection" ||
		err.Error() == "connection reset by peer")
}

func isTimeoutError(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}
	return false
}
