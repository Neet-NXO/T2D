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
	"t2d/internal/tcpopt"

	"github.com/panjf2000/gnet/v2"
)

type upstreamReorderState struct {
	mu          sync.Mutex
	initialized bool
	expected    uint32
	pending     map[uint32][]byte
	lastDrain   time.Time
}

type downstreamWriter struct {
	laneID          int
	batchMaxPackets int
	batchMaxBytes   int
	batchMaxDelay   time.Duration

	server *Server
	ch     chan []byte

	connMu sync.RWMutex
	conn   gnet.Conn
}

func newDownstreamWriter(server *Server, laneID int, packets, bytes int, delay time.Duration) *downstreamWriter {
	w := &downstreamWriter{
		laneID:          laneID,
		batchMaxPackets: max(1, packets),
		batchMaxBytes:   max(1, bytes),
		batchMaxDelay:   max(0, delay),
		server:          server,
		ch:              make(chan []byte, 32768),
	}
	go w.run(server.ctx)
	return w
}

func (w *downstreamWriter) setConn(conn gnet.Conn) {
	w.connMu.Lock()
	w.conn = conn
	w.connMu.Unlock()
}

func (w *downstreamWriter) clearConn(conn gnet.Conn) {
	w.connMu.Lock()
	if w.conn == conn {
		w.conn = nil
	}
	w.connMu.Unlock()
}

func (w *downstreamWriter) getConn() gnet.Conn {
	w.connMu.RLock()
	defer w.connMu.RUnlock()
	return w.conn
}

func (w *downstreamWriter) enqueue(frame []byte) bool {
	select {
	case w.ch <- frame:
		return true
	default:
		return false
	}
}

func (w *downstreamWriter) adaptiveDelay() time.Duration {
	if w.batchMaxDelay <= 0 {
		return 0
	}
	pending := len(w.ch)
	switch {
	case pending >= w.batchMaxPackets:
		return 0
	case pending >= w.batchMaxPackets/2:
		return w.batchMaxDelay / 4
	case pending >= w.batchMaxPackets/4:
		return w.batchMaxDelay / 2
	default:
		return w.batchMaxDelay
	}
}

func (w *downstreamWriter) run(ctx context.Context) {
	batch := make([][]byte, 0, w.batchMaxPackets)
	batchBytes := 0
	flush := func() {
		if len(batch) == 0 {
			return
		}
		conn := w.getConn()
		if conn == nil {
			for _, frame := range batch {
				_ = frame
			}
			batch = batch[:0]
			batchBytes = 0
			return
		}
		frames := append([][]byte(nil), batch...)
		if err := conn.AsyncWritev(frames, nil); err != nil {
			log.Printf("[ERROR] Downstream lane %d async writev error: %v", w.laneID, err)
			w.server.stats.AddError()
		}
		for _, frame := range batch {
			w.server.stats.AddBytesSent(uint64(len(frame)))
			w.server.stats.AddPacketsSent(1)
		}
		batch = batch[:0]
		batchBytes = 0
	}

	for {
		select {
		case <-ctx.Done():
			return
		case frame := <-w.ch:
			batch = append(batch, frame)
			batchBytes += len(frame)

			delay := w.adaptiveDelay()
			if len(batch) < w.batchMaxPackets && batchBytes < w.batchMaxBytes && delay > 0 {
				t := time.NewTimer(delay)
				select {
				case <-ctx.Done():
					t.Stop()
					flush()
					return
				case <-t.C:
				}
			}

			for len(batch) < w.batchMaxPackets && batchBytes < w.batchMaxBytes {
				select {
				case frame = <-w.ch:
					batch = append(batch, frame)
					batchBytes += len(frame)
				default:
					flush()
					goto next
				}
			}
			flush()
		next:
		}
	}
}

// Server 服务端结构
type Server struct {
	config             *config.Config
	sessions           sync.Map // map[uint32]*session.BackendSession
	upstreamEngine     gnet.Engine
	downstreamEngine   gnet.Engine
	upstreamRunning    atomic.Bool
	downstreamRunning  atomic.Bool
	ctx                context.Context
	cancel             context.CancelFunc
	stats              *stats.Stats
	upstreamPayloads   sync.Pool
	maxUpstreamPayload int
	upstreamCipher     crypto.Cipher // 上行解密器
	downstreamCipher   crypto.Cipher // 下行加密器

	downWriters           []*downstreamWriter
	sessionLaneMap        sync.Map // map[uint32]int
	upstreamReorderStates sync.Map // map[uint32]*upstreamReorderState
	upstreamStripeEnabled bool
	upstreamReorderWindow int
	upstreamReorderHold   time.Duration
}

// New 创建新的服务端实例
func New(cfg *config.Config) (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())

	upstreamCryptoConfig := &crypto.Config{
		Type:     crypto.CipherType(cfg.UpstreamCrypto.Type),
		Password: cfg.UpstreamCrypto.Password,
	}
	upstreamCipher, err := crypto.NewCipher(upstreamCryptoConfig)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create upstream cipher: %w", err)
	}

	downstreamCryptoConfig := &crypto.Config{
		Type:     crypto.CipherType(cfg.DownstreamCrypto.Type),
		Password: cfg.DownstreamCrypto.Password,
	}
	downstreamCipher, err := crypto.NewCipher(downstreamCryptoConfig)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create downstream cipher: %w", err)
	}

	maxUpstreamPayload := cfg.MaxPacketSize + upstreamCipher.Overhead()
	s := &Server{
		config:                cfg,
		ctx:                   ctx,
		cancel:                cancel,
		stats:                 stats.New(),
		maxUpstreamPayload:    maxUpstreamPayload,
		upstreamCipher:        upstreamCipher,
		downstreamCipher:      downstreamCipher,
		upstreamStripeEnabled: cfg.UpstreamStripeEnabled,
		upstreamReorderWindow: cfg.UpstreamReorderWindow,
		upstreamReorderHold:   time.Duration(cfg.UpstreamReorderHoldMs) * time.Millisecond,
	}
	s.upstreamPayloads.New = func() interface{} {
		return make([]byte, maxUpstreamPayload)
	}

	downLaneCount := max(1, cfg.DownstreamLaneCount)
	s.downWriters = make([]*downstreamWriter, downLaneCount)
	for i := 0; i < downLaneCount; i++ {
		s.downWriters[i] = newDownstreamWriter(
			s,
			i,
			cfg.DownstreamBatchMaxPackets,
			cfg.DownstreamBatchMaxBytes,
			time.Duration(cfg.DownstreamBatchMaxDelayMs)*time.Millisecond,
		)
	}

	return s, nil
}

func (s *Server) setDownstreamConnForLane(laneID int, conn gnet.Conn) {
	if laneID < 0 || laneID >= len(s.downWriters) {
		return
	}
	s.downWriters[laneID].setConn(conn)
}

func (s *Server) clearDownstreamConnForLane(laneID int, conn gnet.Conn) {
	if laneID < 0 || laneID >= len(s.downWriters) {
		return
	}
	s.downWriters[laneID].clearConn(conn)
}

func (s *Server) laneForSession(sessionID uint32) int {
	if v, ok := s.sessionLaneMap.Load(sessionID); ok {
		if laneID, ok2 := v.(int); ok2 && laneID >= 0 && laneID < len(s.downWriters) {
			return laneID
		}
	}
	return int(sessionID % uint32(len(s.downWriters)))
}

// Start 启动服务端
func (s *Server) Start() error {
	log.Printf("[INFO] Server starting")

	go s.startUpstreamServer()
	go s.startDownstreamServer()
	go s.printStats()
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
	s.closeSessions()

	return nil
}

// Run 运行服务端（阻塞直到收到信号）
func (s *Server) Run() error {
	if err := s.Start(); err != nil {
		return err
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	return s.Stop()
}

func (s *Server) startUpstreamServer() {
	handler := &UpstreamHandler{server: s}

	options := []gnet.Option{
		gnet.WithMulticore(true),
		gnet.WithReusePort(true),
		gnet.WithTCPKeepAlive(time.Duration(config.DefaultTCPKeepAliveSeconds) * time.Second),
		gnet.WithTCPNoDelay(gnet.TCPNoDelay),
		gnet.WithReadBufferCap(config.DefaultGnetReadBufferCap),
		gnet.WithWriteBufferCap(config.DefaultGnetWriteBufferCap),
	}

	err := gnet.Run(handler, "tcp://"+s.config.UpstreamPort, options...)
	if err != nil {
		log.Fatalf("[FATAL] Upstream server error: %v", err)
	}
}

func (s *Server) startDownstreamServer() {
	handler := &DownstreamHandler{server: s}

	options := []gnet.Option{
		gnet.WithMulticore(true),
		gnet.WithReusePort(true),
		gnet.WithTCPKeepAlive(time.Duration(config.DefaultTCPKeepAliveSeconds) * time.Second),
		gnet.WithTCPNoDelay(gnet.TCPNoDelay),
		gnet.WithReadBufferCap(config.DefaultGnetReadBufferCap),
		gnet.WithWriteBufferCap(config.DefaultGnetWriteBufferCap),
	}

	err := gnet.Run(handler, "tcp://"+s.config.DownstreamPort, options...)
	if err != nil {
		log.Fatalf("[FATAL] Downstream server error: %v", err)
	}
}

func (s *Server) handleUpstreamPacket(header *protocol.PacketHeader, data []byte) {
	s.stats.AddPacketsReceived(1)
	s.stats.AddBytesReceived(uint64(protocol.HeaderSize + len(data)))

	decryptedData, err := s.upstreamCipher.Decrypt(data)
	if err != nil {
		log.Printf("[ERROR] Failed to decrypt upstream data: %v", err)
		s.stats.AddError()
		return
	}

	var sess *session.BackendSession
	if v, ok := s.sessions.Load(header.SessionID); ok {
		sess = v.(*session.BackendSession)
		sess.UpdateLastActive()
	} else {
		backendAddr, err := net.ResolveUDPAddr("udp", s.config.BackendAddr)
		if err != nil {
			log.Printf("[ERROR] Failed to resolve backend address: %v", err)
			s.stats.AddError()
			return
		}

		backendNetwork := "udp"
		if backendAddr.IP != nil && backendAddr.IP.To4() != nil {
			backendNetwork = "udp4"
		}
		udpConn, err := net.DialUDP(backendNetwork, nil, backendAddr)
		if err != nil {
			log.Printf("[ERROR] Failed to dial backend: %v", err)
			s.stats.AddError()
			return
		}
		if err := udpConn.SetReadBuffer(config.DefaultSocketBuffer); err != nil {
			log.Printf("[WARN] Failed to set backend UDP read buffer: %v", err)
		}
		if err := udpConn.SetWriteBuffer(config.DefaultSocketBuffer); err != nil {
			log.Printf("[WARN] Failed to set backend UDP write buffer: %v", err)
		}

		sess = session.NewBackendSession(header.SessionID, udpConn, backendAddr, nil)
		s.sessions.Store(header.SessionID, sess)
		s.stats.AddActiveSession()
		go s.handleBackendData(sess)
		log.Printf("[INFO] New backend session %d to %s", header.SessionID, backendAddr)
	}

	s.writeUpstreamToBackend(sess, header.SequenceID, decryptedData)
}

func (s *Server) writeUpstreamToBackend(sess *session.BackendSession, seq uint32, payload []byte) {
	if !s.upstreamStripeEnabled {
		s.writeBackendPacket(sess, payload)
		return
	}

	v, _ := s.upstreamReorderStates.LoadOrStore(sess.GetSessionID(), &upstreamReorderState{
		pending: make(map[uint32][]byte),
	})
	state := v.(*upstreamReorderState)
	now := time.Now()

	state.mu.Lock()
	defer state.mu.Unlock()

	if !state.initialized {
		state.initialized = true
		state.expected = seq
		state.lastDrain = now
	}

	// 旧序列直接丢弃，避免重复注入。
	if seq < state.expected {
		return
	}

	if seq == state.expected {
		if !s.writeBackendPacket(sess, payload) {
			return
		}
		state.expected++
		state.lastDrain = now
		for {
			next, ok := state.pending[state.expected]
			if !ok {
				break
			}
			delete(state.pending, state.expected)
			if !s.writeBackendPacket(sess, next) {
				return
			}
			state.expected++
			state.lastDrain = now
		}
		return
	}

	// 超窗口表示乱序过大，直接重置窗口到当前包，避免无限等待。
	if seq-state.expected > uint32(max(1, s.upstreamReorderWindow)) {
		clear(state.pending)
		if s.writeBackendPacket(sess, payload) {
			state.expected = seq + 1
			state.lastDrain = now
		}
		return
	}

	if _, exists := state.pending[seq]; !exists {
		cp := append([]byte(nil), payload...)
		state.pending[seq] = cp
	}

	if len(state.pending) >= max(1, s.upstreamReorderWindow) || now.Sub(state.lastDrain) >= s.upstreamReorderHold {
		s.forceDrainReorderLocked(state, sess, now)
	}
}

func (s *Server) forceDrainReorderLocked(state *upstreamReorderState, sess *session.BackendSession, now time.Time) {
	if len(state.pending) == 0 {
		return
	}

	minSeq := uint32(0)
	found := false
	for seq := range state.pending {
		if !found || seq < minSeq {
			minSeq = seq
			found = true
		}
	}
	if !found {
		return
	}

	state.expected = minSeq
	for {
		payload, ok := state.pending[state.expected]
		if !ok {
			break
		}
		delete(state.pending, state.expected)
		if !s.writeBackendPacket(sess, payload) {
			return
		}
		state.expected++
		state.lastDrain = now
	}
}

func (s *Server) writeBackendPacket(sess *session.BackendSession, payload []byte) bool {
	if _, err := sess.GetConn().Write(payload); err != nil {
		log.Printf("[ERROR] Backend write error: %v", err)
		s.stats.AddError()
		return false
	}
	s.stats.AddBytesSent(uint64(len(payload)))
	s.stats.AddPacketsSent(1)
	return true
}

func (s *Server) handleBackendData(sess *session.BackendSession) {
	defer func() {
		sess.Close()
		if _, ok := s.sessions.LoadAndDelete(sess.GetSessionID()); ok {
			s.sessionLaneMap.Delete(sess.GetSessionID())
			s.upstreamReorderStates.Delete(sess.GetSessionID())
			s.stats.RemoveActiveSession()
		}
		log.Printf("[INFO] Backend session %d closed", sess.GetSessionID())
	}()

	buf := make([]byte, s.config.MaxPacketSize)
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

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

		s.sendToDownstream(sess, buf[:n])
	}
}

func (s *Server) sendToDownstream(sess *session.BackendSession, data []byte) {
	laneID := s.laneForSession(sess.GetSessionID())
	if laneID < 0 || laneID >= len(s.downWriters) {
		log.Printf("[WARN] Invalid downstream lane for session %d", sess.GetSessionID())
		return
	}

	encryptedData, err := s.downstreamCipher.Encrypt(data)
	if err != nil {
		log.Printf("[ERROR] Failed to encrypt downstream data: %v", err)
		s.stats.AddError()
		return
	}

	sequenceID := sess.NextSequenceID()
	header := protocol.NewDataPacket(sess.GetSessionID(), sequenceID, uint32(len(encryptedData)))
	frame := make([]byte, protocol.HeaderSize+len(encryptedData))
	header.MarshalTo(frame[:protocol.HeaderSize])
	copy(frame[protocol.HeaderSize:], encryptedData)

	if ok := s.downWriters[laneID].enqueue(frame); !ok {
		log.Printf("[WARN] Downstream lane %d queue full, dropping session %d", laneID, sess.GetSessionID())
		s.stats.AddError()
	}
}

func (s *Server) applyConnOptions(conn gnet.Conn, direction tcpopt.Direction) {
	tcpopt.WarnIfUnsupported(tcpopt.ApplyGnetConn(conn), "server "+string(direction))
}

func (s *Server) maxUpstreamFramePayload() int {
	return s.maxUpstreamPayload
}

func (s *Server) maxDownstreamControlPayload() int {
	return 64 * 1024
}

func (s *Server) getUpstreamPayload(size int) []byte {
	if size <= 0 {
		return nil
	}
	if s.maxUpstreamPayload > 0 && size <= s.maxUpstreamPayload {
		buf := s.upstreamPayloads.Get().([]byte)
		return buf[:size]
	}
	return make([]byte, size)
}

func (s *Server) putUpstreamPayload(buf []byte) {
	if buf == nil {
		return
	}
	if s.maxUpstreamPayload > 0 && cap(buf) == s.maxUpstreamPayload {
		s.upstreamPayloads.Put(buf[:s.maxUpstreamPayload])
	}
}

func (s *Server) closeSessions() {
	s.sessions.Range(func(key, value interface{}) bool {
		if sess, ok := value.(*session.BackendSession); ok {
			sess.Close()
		}
		return true
	})
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
					s.sessionLaneMap.Delete(sessionID)
					s.upstreamReorderStates.Delete(sessionID)
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
