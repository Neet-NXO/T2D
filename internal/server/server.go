package server

import (
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

	"github.com/panjf2000/gnet/v2"
	"github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
)

// Server 服务端结构
type Server struct {
	config            *config.Config
	sessions          sync.Map // map[uint32]*session.BackendSession
	upstreamEngine    gnet.Engine
	downstreamEngine  gnet.Engine
	upstreamRunning   atomic.Bool
	downstreamRunning atomic.Bool
	connMap           sync.Map // map[gnet.Conn]uint32 连接到会话的映射
	ctx               context.Context
	cancel            context.CancelFunc
	stats             *stats.Stats
	pool              *goroutine.Pool
	upstreamCipher    crypto.Cipher // 上行解密器（对应客户端的上行加密）
	downstreamCipher  crypto.Cipher // 下行加密器（对应客户端的下行解密）
}

// New 创建新的服务端实例
func New(cfg *config.Config) (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建协程池
	pool := goroutine.Default()

	// 创建上行解密器（对应客户端的上行加密）
	upstreamCryptoConfig := &crypto.Config{
		Type:     crypto.CipherType(cfg.UpstreamCrypto.Type),
		Password: cfg.UpstreamCrypto.Password,
	}
	upstreamCipher, err := crypto.NewCipher(upstreamCryptoConfig)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create upstream cipher: %w", err)
	}

	// 创建下行加密器（对应客户端的下行解密）
	downstreamCryptoConfig := &crypto.Config{
		Type:     crypto.CipherType(cfg.DownstreamCrypto.Type),
		Password: cfg.DownstreamCrypto.Password,
	}
	downstreamCipher, err := crypto.NewCipher(downstreamCryptoConfig)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create downstream cipher: %w", err)
	}

	return &Server{
		config:           cfg,
		ctx:              ctx,
		cancel:           cancel,
		stats:            stats.New(),
		pool:             pool,
		upstreamCipher:   upstreamCipher,
		downstreamCipher: downstreamCipher,
	}, nil
}

// Start 启动服务端
func (s *Server) Start() error {
	log.Printf("[INFO] Server starting")

	// 启动上行TCP服务器
	go s.startUpstreamServer()

	// 启动下行TCP服务器
	go s.startDownstreamServer()

	// 启动统计输出
	go s.printStats()

	// 启动会话清理
	go s.cleanupSessions()

	return nil
}

// Stop 停止服务端
func (s *Server) Stop() error {
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

// Run 运行服务端（阻塞直到收到信号）
func (s *Server) Run() error {
	if err := s.Start(); err != nil {
		return err
	}

	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	return s.Stop()
}

func (s *Server) startUpstreamServer() {
	handler := &UpstreamHandler{
		server: s,
		pool:   s.pool,
	}

	options := []gnet.Option{
		gnet.WithMulticore(s.config.Multicore),
		gnet.WithReusePort(s.config.ReusePort),
		gnet.WithTCPKeepAlive(time.Duration(s.config.TCPKeepAlive) * time.Second),
		gnet.WithReadBufferCap(s.config.ReadBufferCap),
		gnet.WithWriteBufferCap(s.config.WriteBufferCap),
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
		gnet.WithMulticore(s.config.Multicore),
		gnet.WithReusePort(s.config.ReusePort),
		gnet.WithTCPKeepAlive(time.Duration(s.config.TCPKeepAlive) * time.Second),
		gnet.WithReadBufferCap(s.config.ReadBufferCap),
		gnet.WithWriteBufferCap(s.config.WriteBufferCap),
	}

	err := gnet.Run(handler, "tcp://"+s.config.DownstreamPort, options...)
	if err != nil {
		log.Fatalf("[FATAL] Downstream server error: %v", err)
	}
}

func (s *Server) handleUpstreamPacket(conn gnet.Conn, header *protocol.PacketHeader, data []byte) {
	s.stats.AddPacketsReceived(1)
	s.stats.AddBytesReceived(uint64(protocol.HeaderSize + len(data)))

	// 解密上行数据
	decryptedData, err := s.upstreamCipher.Decrypt(data)
	if err != nil {
		log.Printf("[ERROR] Failed to decrypt upstream data: %v", err)
		s.stats.AddError()
		return
	}

	// 查找或创建后端会话
	var sess *session.BackendSession
	if v, ok := s.sessions.Load(header.SessionID); ok {
		sess = v.(*session.BackendSession)
		sess.UpdateLastActive()
	} else {
		// 创建新会话
		backendAddr, err := net.ResolveUDPAddr("udp", s.config.BackendAddr)
		if err != nil {
			log.Printf("[ERROR] Failed to resolve backend address: %v", err)
			s.stats.AddError()
			return
		}

		udpConn, err := net.DialUDP("udp", nil, backendAddr)
		if err != nil {
			log.Printf("[ERROR] Failed to dial backend: %v", err)
			s.stats.AddError()
			return
		}

		udpConn.SetReadBuffer(s.config.SocketRecvBuffer)
		udpConn.SetWriteBuffer(s.config.SocketSendBuffer)

		sess = session.NewBackendSession(header.SessionID, udpConn, backendAddr, nil)
		s.sessions.Store(header.SessionID, sess)
		s.stats.AddActiveSession()

		// 启动后端数据接收
		go s.handleBackendData(sess)

		log.Printf("[INFO] New backend session %d to %s", header.SessionID, backendAddr)
	}

	// 发送解密后的数据到后端
	if _, err := sess.GetConn().Write(decryptedData); err != nil {
		log.Printf("[ERROR] Backend write error: %v", err)
		s.stats.AddError()
		return
	}

	s.stats.AddBytesSent(uint64(len(decryptedData)))
	s.stats.AddPacketsSent(1)
}

func (s *Server) handleBackendData(sess *session.BackendSession) {
	defer func() {
		sess.Close()
		s.sessions.Delete(sess.GetSessionID())
		s.stats.RemoveActiveSession()
		log.Printf("[INFO] Backend session %d closed", sess.GetSessionID())
	}()

	buf := make([]byte, s.config.MaxPacketSize)
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		sess.GetConn().SetReadDeadline(time.Now().Add(30 * time.Second))
		n, err := sess.GetConn().Read(buf)
		if err != nil {
			if !isTimeoutError(err) && !isClosedError(err) {
				log.Printf("[ERROR] Backend read error: %v", err)
				s.stats.AddError()
			}
			return
		}

		s.stats.AddBytesReceived(uint64(n))
		s.stats.AddPacketsReceived(1)
		sess.UpdateLastActive()

		// 发送到下行连接
		data := make([]byte, n)
		copy(data, buf[:n])

		s.pool.Submit(func() {
			s.sendToDownstream(sess, data)
		})
	}
}

func (s *Server) sendToDownstream(sess *session.BackendSession, data []byte) {
	downstreamConn := sess.GetDownstreamConn()
	if downstreamConn == nil {
		log.Printf("[WARN] No downstream connection for session %d", sess.GetSessionID())
		return
	}

	// 加密下行数据
	encryptedData, err := s.downstreamCipher.Encrypt(data)
	if err != nil {
		log.Printf("[ERROR] Failed to encrypt downstream data: %v", err)
		s.stats.AddError()
		return
	}

	sequenceID := sess.NextSequenceID()
	header := protocol.NewDataPacket(sess.GetSessionID(), sequenceID, uint32(len(encryptedData)))

	packet := append(header.Marshal(), encryptedData...)
	if err := downstreamConn.AsyncWrite(packet, nil); err != nil {
		log.Printf("[ERROR] Downstream write error: %v", err)
		s.stats.AddError()
		return
	}

	s.stats.AddBytesSent(uint64(len(packet)))
	s.stats.AddPacketsSent(1)
}

func (s *Server) cleanupSessions() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			timeout := time.Duration(s.config.SessionTimeout) * time.Second
			var expiredSessions []uint32

			s.sessions.Range(func(key, value interface{}) bool {
				sess := value.(*session.BackendSession)
				if sess.IsExpired(timeout) {
					expiredSessions = append(expiredSessions, sess.GetSessionID())
				}
				return true
			})

			for _, sessionID := range expiredSessions {
				if v, ok := s.sessions.LoadAndDelete(sessionID); ok {
					sess := v.(*session.BackendSession)
					sess.Close()
					log.Printf("[INFO] Cleaning up expired session %d", sessionID)
					s.stats.RemoveActiveSession()
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
			summary := s.stats.GetSummary()
			log.Printf("[STATS] Server - Sessions: %d, Sent: %d bytes/%d packets, Received: %d bytes/%d packets, Errors: %d, Uptime: %v",
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
