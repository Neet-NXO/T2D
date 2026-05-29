package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/netip"
	"os"
	"os/signal"
	"slices"
	"strings"
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

	"github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
)

type upstreamLane struct {
	id        int
	ring      *upstreamRing
	reconnect chan struct{}

	mu   sync.Mutex
	conn net.Conn
}

type downstreamLane struct {
	id         int
	reconnect  chan struct{}
	registerCh chan uint32

	mu   sync.Mutex
	conn net.Conn
}

// Client 客户端结构
type Client struct {
	config            *config.Config
	udpListener       *net.UDPConn
	sessions          sync.Map // map[netip.AddrPort]*session.UDPSession
	sessionsByID      sync.Map // map[uint32]*session.UDPSession
	sessionCounter    uint32
	maxUpstreamPacket int

	upstreamLanes   []*upstreamLane
	downstreamLanes []*downstreamLane

	upstreamBatchMaxPackets int
	upstreamBatchMaxBytes   int
	upstreamBatchMaxDelay   time.Duration
	upstreamStripeEnabled   bool
	downstreamHeartbeat     time.Duration
	upstreamZeroCopy        bool

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

	pool := goroutine.Default()

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

	maxUpstreamPacket := protocol.HeaderSize + cfg.MaxPacketSize + upstreamCipher.Overhead()
	// Multi-lane path currently uses per-lane ring scheduling; keep UDP read path allocation-stable.
	upstreamZeroCopy := false

	upLaneCount := max(1, cfg.UpstreamLaneCount)
	downLaneCount := max(1, cfg.DownstreamLaneCount)
	upstreamLanes := make([]*upstreamLane, 0, upLaneCount)
	for i := 0; i < upLaneCount; i++ {
		upstreamLanes = append(upstreamLanes, &upstreamLane{
			id:        i,
			ring:      newUpstreamRing(cfg.UpstreamQueueSize, maxUpstreamPacket),
			reconnect: make(chan struct{}, 1),
		})
	}
	downstreamLanes := make([]*downstreamLane, 0, downLaneCount)
	for i := 0; i < downLaneCount; i++ {
		downstreamLanes = append(downstreamLanes, &downstreamLane{
			id:         i,
			reconnect:  make(chan struct{}, 1),
			registerCh: make(chan uint32, 4096),
		})
	}

	client := &Client{
		config:                  cfg,
		ctx:                     ctx,
		cancel:                  cancel,
		stats:                   stats.New(),
		pool:                    pool,
		maxUpstreamPacket:       maxUpstreamPacket,
		upstreamLanes:           upstreamLanes,
		downstreamLanes:         downstreamLanes,
		upstreamBatchMaxPackets: cfg.UpstreamBatchMaxPackets,
		upstreamBatchMaxBytes:   cfg.UpstreamBatchMaxBytes,
		upstreamBatchMaxDelay:   time.Duration(cfg.UpstreamBatchMaxDelayMs) * time.Millisecond,
		upstreamStripeEnabled:   cfg.UpstreamStripeEnabled,
		downstreamHeartbeat:     time.Duration(cfg.HeartbeatIntervalMs) * time.Millisecond,
		upstreamZeroCopy:        upstreamZeroCopy,
		upstreamCipher:          upstreamCipher,
		downstreamCipher:        downstreamCipher,
	}
	return client, nil
}

// Start 启动客户端
func (c *Client) Start() error {
	udpAddr, err := net.ResolveUDPAddr("udp", c.config.ListenAddr)
	if err != nil {
		return err
	}

	udpNetwork := "udp"
	if udpAddr.IP != nil && udpAddr.IP.To4() != nil {
		udpNetwork = "udp4"
	}
	c.udpListener, err = net.ListenUDP(udpNetwork, udpAddr)
	if err != nil {
		return err
	}
	if err := c.udpListener.SetReadBuffer(config.DefaultSocketBuffer); err != nil {
		log.Printf("[WARN] Failed to set client UDP read buffer: %v", err)
	}
	if err := c.udpListener.SetWriteBuffer(config.DefaultSocketBuffer); err != nil {
		log.Printf("[WARN] Failed to set client UDP write buffer: %v", err)
	}

	log.Printf("[INFO] Client listening on UDP %s", c.config.ListenAddr)

	for _, lane := range c.upstreamLanes {
		go c.handleUpstreamWrites(lane)
		go c.manageUpstreamConnection(lane)
	}
	for _, lane := range c.downstreamLanes {
		go c.manageDownstreamConnection(lane)
	}

	go c.handleUDP()
	go c.cleanupSessions()
	go c.printStats()

	return nil
}

// Stop 停止客户端
func (c *Client) Stop() error {
	log.Println("[INFO] Shutting down client...")
	c.cancel()

	for _, lane := range c.upstreamLanes {
		lane.ring.close()
		lane.mu.Lock()
		if lane.conn != nil {
			lane.conn.Close()
			lane.conn = nil
		}
		lane.mu.Unlock()
	}
	for _, lane := range c.downstreamLanes {
		lane.mu.Lock()
		if lane.conn != nil {
			lane.conn.Close()
			lane.conn = nil
		}
		lane.mu.Unlock()
	}

	if c.udpListener != nil {
		c.udpListener.Close()
	}

	return nil
}

// Run 运行客户端（阻塞直到收到信号）
func (c *Client) Run() error {
	if err := c.Start(); err != nil {
		return err
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	return c.Stop()
}

func (c *Client) upstreamLaneForSession(sessionID uint32) *upstreamLane {
	idx := int(sessionID % uint32(len(c.upstreamLanes)))
	return c.upstreamLanes[idx]
}

func (c *Client) selectUpstreamLane(sessionID, sequenceID uint32) *upstreamLane {
	if c.upstreamStripeEnabled {
		idx := int(sequenceID % uint32(len(c.upstreamLanes)))
		return c.upstreamLanes[idx]
	}
	return c.upstreamLaneForSession(sessionID)
}

func (c *Client) downstreamLaneForSession(sessionID uint32) *downstreamLane {
	idx := int(sessionID % uint32(len(c.downstreamLanes)))
	return c.downstreamLanes[idx]
}

func (c *Client) manageUpstreamConnection(lane *upstreamLane) {
	attempt := 0
	for c.ctx.Err() == nil {
		if c.getUpstreamConn(lane) != nil {
			select {
			case <-c.ctx.Done():
				return
			case <-lane.reconnect:
				continue
			}
		}

		log.Printf("[INFO] Connecting upstream lane %d to %s", lane.id, c.config.ServerUpstream)
		conn, err := c.dialTCP(c.config.ServerUpstream, tcpopt.DirectionUpstream)
		if err != nil {
			delay := c.reconnectDelay(attempt)
			attempt++
			log.Printf("[ERROR] Failed to connect upstream lane %d: %v; retrying in %v", lane.id, err, delay)
			if !c.waitReconnectDelay(delay, lane.reconnect) {
				return
			}
			continue
		}

		attempt = 0
		lane.mu.Lock()
		lane.conn = conn
		lane.mu.Unlock()
		go c.watchUpstreamConn(lane, conn)
		c.drainReconnect(lane.reconnect)
		log.Printf("[INFO] Connected upstream lane %d to %s", lane.id, c.config.ServerUpstream)
	}
}

func (c *Client) watchUpstreamConn(lane *upstreamLane, conn net.Conn) {
	var b [1]byte
	_, err := conn.Read(b[:])
	if c.ctx.Err() != nil {
		return
	}
	lane.mu.Lock()
	if lane.conn == conn {
		lane.conn.Close()
		lane.conn = nil
	}
	lane.mu.Unlock()
	if err != nil && !errors.Is(err, io.EOF) && !isClosedError(err) {
		log.Printf("[WARN] Upstream lane %d passive close: %v", lane.id, err)
	}
	c.notifyReconnect(lane.reconnect)
}

func (c *Client) manageDownstreamConnection(lane *downstreamLane) {
	attempt := 0
	for c.ctx.Err() == nil {
		if c.getDownstreamConn(lane) != nil {
			select {
			case <-c.ctx.Done():
				return
			case <-lane.reconnect:
				continue
			}
		}

		log.Printf("[INFO] Connecting downstream lane %d to %s", lane.id, c.config.ServerDownstream)
		conn, err := c.dialTCP(c.config.ServerDownstream, tcpopt.DirectionDownstream)
		if err != nil {
			delay := c.reconnectDelay(attempt)
			attempt++
			log.Printf("[ERROR] Failed to connect downstream lane %d: %v; retrying in %v", lane.id, err, delay)
			if !c.waitReconnectDelay(delay, lane.reconnect) {
				return
			}
			continue
		}

		attempt = 0
		lane.mu.Lock()
		lane.conn = conn
		lane.mu.Unlock()
		c.drainReconnect(lane.reconnect)
		log.Printf("[INFO] Connected downstream lane %d to %s", lane.id, c.config.ServerDownstream)

		c.registerAllSessionsForLane(lane)
		c.handleDownstream(lane, conn)
	}
}

func (c *Client) dialTCP(addr string, direction tcpopt.Direction) (net.Conn, error) {
	dialer := net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: time.Duration(config.DefaultTCPKeepAliveSeconds) * time.Second,
	}
	conn, err := dialer.DialContext(c.ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpopt.WarnIfUnsupported(tcpopt.ApplyTCPConn(tcpConn), "client "+string(direction))
		if err := tcpConn.SetKeepAlive(true); err != nil {
			log.Printf("[WARN] Failed to enable TCP keepalive for client %s: %v", direction, err)
		}
		if err := tcpConn.SetKeepAlivePeriod(time.Duration(config.DefaultTCPKeepAliveSeconds) * time.Second); err != nil {
			log.Printf("[WARN] Failed to set TCP keepalive period for client %s: %v", direction, err)
		}
	}
	return conn, nil
}

func (c *Client) getUpstreamConn(lane *upstreamLane) net.Conn {
	lane.mu.Lock()
	defer lane.mu.Unlock()
	return lane.conn
}

func (c *Client) getDownstreamConn(lane *downstreamLane) net.Conn {
	lane.mu.Lock()
	defer lane.mu.Unlock()
	return lane.conn
}

func (c *Client) reconnectDelay(attempt int) time.Duration {
	base := time.Duration(c.config.ReconnectInterval) * time.Second
	if base <= 0 {
		base = time.Second
	}
	maxDelay := time.Duration(c.config.ReconnectMaxInterval) * time.Second
	if maxDelay < base {
		maxDelay = base
	}

	delay := base
	for i := 0; i < attempt && delay < maxDelay; i++ {
		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}
	if c.config.ReconnectJitterMs > 0 {
		jitterMax := int64(c.config.ReconnectJitterMs) * int64(time.Millisecond)
		delay += time.Duration(rand.Int63n(jitterMax + 1))
	}
	return delay
}

func (c *Client) waitReconnectDelay(delay time.Duration, wake <-chan struct{}) bool {
	if delay <= 0 {
		return c.ctx.Err() == nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-c.ctx.Done():
		return false
	case <-wake:
		return true
	case <-timer.C:
		return true
	}
}

func (c *Client) drainReconnect(ch <-chan struct{}) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

func (c *Client) notifyReconnect(ch chan<- struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

func (c *Client) handleUDP() {
	if c.upstreamZeroCopy {
		c.handleUDPZeroCopy()
		return
	}

	buf := make([]byte, c.config.MaxPacketSize)
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		n, addr, err := c.udpListener.ReadFromUDPAddrPort(buf)
		if err != nil {
			if !isClosedError(err) {
				log.Printf("[ERROR] UDP read error: %v", err)
				c.stats.AddError()
			}
			continue
		}

		c.stats.AddBytesReceived(uint64(n))
		c.stats.AddPacketsReceived(1)

		sess := c.getOrCreateSession(addr)
		c.sendToUpstream(sess, buf[:n])
	}
}

func (c *Client) handleUDPZeroCopy() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		// 首次读取无法提前确定lane，先读入临时缓冲后按会话lane转存。
		buf := make([]byte, c.config.MaxPacketSize)
		n, addr, err := c.udpListener.ReadFromUDPAddrPort(buf)
		if err != nil {
			if !isClosedError(err) {
				log.Printf("[ERROR] UDP read error: %v", err)
				c.stats.AddError()
			}
			continue
		}

		c.stats.AddBytesReceived(uint64(n))
		c.stats.AddPacketsReceived(1)

		sess := c.getOrCreateSession(addr)
		sequenceID := sess.NextSequenceID()
		lane := c.selectUpstreamLane(sess.GetSessionID(), sequenceID)
		packet := lane.ring.acquire()
		if packet == nil {
			return
		}
		header := protocol.NewDataPacket(sess.GetSessionID(), sequenceID, uint32(n))
		header.MarshalTo(packet.buf[:protocol.HeaderSize])
		copy(packet.buf[protocol.HeaderSize:], buf[:n])
		packet.data = packet.buf[:protocol.HeaderSize+n]
		if !lane.ring.submit(packet) {
			lane.ring.discard(packet)
			return
		}
	}
}

func (c *Client) getOrCreateSession(addr netip.AddrPort) *session.UDPSession {
	addr = netip.AddrPortFrom(addr.Addr().Unmap(), addr.Port())
	sessionKey := addr
	if v, ok := c.sessions.Load(sessionKey); ok {
		sess := v.(*session.UDPSession)
		sess.UpdateLastActive()
		return sess
	}

	sessionID := atomic.AddUint32(&c.sessionCounter, 1)
	sess := session.NewUDPSessionAddrPort(sessionID, c.udpListener, addr)
	c.sessions.Store(sessionKey, sess)
	c.sessionsByID.Store(sessionID, sess)
	c.stats.AddActiveSession()
	log.Printf("[INFO] New UDP session %d from %s", sessionID, addr)
	c.registerSessionOnLane(sess.GetSessionID())
	return sess
}

func (c *Client) registerAllSessionsForLane(lane *downstreamLane) {
	var ids []uint32
	c.sessionsByID.Range(func(key, value interface{}) bool {
		sid := key.(uint32)
		if c.downstreamLaneForSession(sid).id == lane.id {
			ids = append(ids, sid)
		}
		return true
	})
	if len(ids) == 0 {
		return
	}
	slices.Sort(ids)
	c.sendLaneControl(lane, protocol.ControlSubtypeLaneRegister, ids)
}

func (c *Client) registerSessionOnLane(sessionID uint32) {
	lane := c.downstreamLaneForSession(sessionID)
	select {
	case lane.registerCh <- sessionID:
	default:
		// 队列满时由心跳兜底重新全量注册。
	}
}

func (c *Client) sendToUpstream(sess *session.UDPSession, data []byte) {
	encryptedData, err := c.upstreamCipher.Encrypt(data)
	if err != nil {
		log.Printf("[ERROR] Failed to encrypt upstream data: %v", err)
		c.stats.AddError()
		return
	}

	sequenceID := sess.NextSequenceID()
	header := protocol.NewDataPacket(sess.GetSessionID(), sequenceID, uint32(len(encryptedData)))
	packetLen := protocol.HeaderSize + len(encryptedData)
	lane := c.selectUpstreamLane(sess.GetSessionID(), sequenceID)
	packet := lane.ring.acquire()
	if packet == nil {
		return
	}
	if packetLen > len(packet.buf) {
		log.Printf("[ERROR] Upstream packet too large: %d > %d", packetLen, len(packet.buf))
		c.stats.AddError()
		lane.ring.discard(packet)
		return
	}
	header.MarshalTo(packet.buf[:protocol.HeaderSize])
	copy(packet.buf[protocol.HeaderSize:], encryptedData)
	packet.data = packet.buf[:packetLen]
	if !lane.ring.submit(packet) {
		lane.ring.discard(packet)
	}
}

func (c *Client) adaptiveUpstreamDelay(lane *upstreamLane) time.Duration {
	if c.upstreamBatchMaxDelay <= 0 {
		return 0
	}
	pending := lane.ring.inflight()
	switch {
	case pending >= c.upstreamBatchMaxPackets:
		return 0
	case pending >= c.upstreamBatchMaxPackets/2:
		return c.upstreamBatchMaxDelay / 4
	case pending >= c.upstreamBatchMaxPackets/4:
		return c.upstreamBatchMaxDelay / 2
	default:
		return c.upstreamBatchMaxDelay
	}
}

func (c *Client) handleUpstreamWrites(lane *upstreamLane) {
	batch := make([]*upstreamPacket, 0, c.upstreamBatchMaxPackets)
	writev := make([][]byte, 0, c.upstreamBatchMaxPackets)
	batchBytes := 0
	flush := func() {
		if len(batch) == 0 {
			return
		}
		writev = writev[:0]
		for _, packet := range batch {
			writev = append(writev, packet.data)
		}
		c.writeUpstreamBatch(lane, writev, len(batch))
		for _, packet := range batch {
			lane.ring.release(packet)
		}
		batch = batch[:0]
		batchBytes = 0
	}

	for {
		packet := lane.ring.pop()
		if packet == nil {
			return
		}
		batch = append(batch, packet)
		batchBytes += len(packet.data)

		delay := c.adaptiveUpstreamDelay(lane)
		if len(batch) < c.upstreamBatchMaxPackets && batchBytes < c.upstreamBatchMaxBytes && delay > 0 {
			t := time.NewTimer(delay)
			select {
			case <-c.ctx.Done():
				t.Stop()
				flush()
				return
			case <-t.C:
			}
		}

		for len(batch) < c.upstreamBatchMaxPackets && batchBytes < c.upstreamBatchMaxBytes {
			packet = lane.ring.tryPop()
			if packet == nil {
				break
			}
			batch = append(batch, packet)
			batchBytes += len(packet.data)
		}
		flush()
	}
}

func (c *Client) writeUpstreamBatch(lane *upstreamLane, batch [][]byte, packetCount int) {
	lane.mu.Lock()
	defer lane.mu.Unlock()

	if lane.conn == nil {
		c.stats.AddError()
		return
	}

	buffers := net.Buffers(batch)
	written, err := buffers.WriteTo(lane.conn)
	if written > 0 {
		c.stats.AddBytesSent(uint64(written))
	}
	if err != nil {
		log.Printf("[ERROR] Upstream lane %d write error: %v", lane.id, err)
		c.stats.AddError()
		lane.conn.Close()
		lane.conn = nil
		c.notifyReconnect(lane.reconnect)
		return
	}
	c.stats.AddPacketsSent(uint64(packetCount))
}

func (c *Client) handleDownstream(lane *downstreamLane, conn net.Conn) {
	defer func() {
		conn.Close()
		lane.mu.Lock()
		if lane.conn == conn {
			lane.conn = nil
		}
		lane.mu.Unlock()
		c.notifyReconnect(lane.reconnect)
	}()

	done := make(chan struct{})
	defer close(done)

	go c.downstreamControlLoop(lane, conn, done)

	buf := make([]byte, c.config.BufferSize)
	readBuf := protocol.NewStreamBuffer(c.config.BufferSize, c.config.MaxPacketSize+c.downstreamCipher.Overhead()+2048)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		n, err := conn.Read(buf)
		if err != nil {
			if !isTimeoutError(err) && !isClosedError(err) {
				log.Printf("[ERROR] Downstream lane %d read error: %v", lane.id, err)
				c.stats.AddError()
			}
			return
		}

		readBuf.Write(buf[:n])
		c.stats.AddBytesReceived(uint64(n))

		for {
			frame, ok, err := readBuf.NextRef()
			if err != nil {
				log.Printf("[ERROR] Invalid downstream packet on lane %d: %v", lane.id, err)
				readBuf.Reset()
				break
			}
			if !ok {
				break
			}
			if frame.Header.Type == protocol.PacketTypeControl {
				continue
			}
			c.handleDownstreamPacket(&frame.Header, frame.Payload)
		}
	}
}

func (c *Client) downstreamControlLoop(lane *downstreamLane, conn net.Conn, done <-chan struct{}) {
	if c.downstreamHeartbeat <= 0 {
		c.downstreamHeartbeat = time.Second
	}
	heartbeatTicker := time.NewTicker(c.downstreamHeartbeat)
	defer heartbeatTicker.Stop()
	flushTicker := time.NewTicker(100 * time.Millisecond)
	defer flushTicker.Stop()

	pending := make(map[uint32]struct{})
	flushPending := func() {
		if len(pending) == 0 {
			return
		}
		ids := make([]uint32, 0, len(pending))
		for sid := range pending {
			ids = append(ids, sid)
		}
		clear(pending)
		slices.Sort(ids)
		c.sendLaneControl(lane, protocol.ControlSubtypeLaneRegister, ids)
	}

	c.sendLaneControl(lane, protocol.ControlSubtypeHello, nil)

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-done:
			return
		case sid := <-lane.registerCh:
			pending[sid] = struct{}{}
		case <-flushTicker.C:
			flushPending()
		case <-heartbeatTicker.C:
			flushPending()
			c.sendLaneControl(lane, protocol.ControlSubtypeLaneHeartbeat, nil)
		}
	}
}

func (c *Client) sendLaneControl(lane *downstreamLane, subtype uint8, sessionIDs []uint32) {
	lane.mu.Lock()
	defer lane.mu.Unlock()
	if lane.conn == nil {
		return
	}

	ctrl := &protocol.LaneControlPayload{
		Subtype:    subtype,
		LaneID:     uint16(lane.id),
		LaneCount:  uint16(len(c.downstreamLanes)),
		SessionIDs: sessionIDs,
	}
	header, payload, err := protocol.NewControlPacketPayload(0, ctrl)
	if err != nil {
		log.Printf("[ERROR] Build lane control failed lane=%d subtype=%d: %v", lane.id, subtype, err)
		c.stats.AddError()
		return
	}
	frame := make([]byte, protocol.HeaderSize+len(payload))
	header.MarshalTo(frame[:protocol.HeaderSize])
	copy(frame[protocol.HeaderSize:], payload)

	if _, err := lane.conn.Write(frame); err != nil {
		log.Printf("[ERROR] Downstream lane %d control write error: %v", lane.id, err)
		c.stats.AddError()
		lane.conn.Close()
		lane.conn = nil
		c.notifyReconnect(lane.reconnect)
	}
}

func (c *Client) handleDownstreamPacket(header *protocol.PacketHeader, data []byte) {
	if header.Type == protocol.PacketTypeHeartbeat {
		return
	}

	decryptedData, err := c.downstreamCipher.Decrypt(data)
	if err != nil {
		log.Printf("[ERROR] Failed to decrypt downstream data: %v", err)
		c.stats.AddError()
		return
	}

	value, ok := c.sessionsByID.Load(header.SessionID)
	if !ok {
		log.Printf("[WARN] Session %d not found for downstream packet", header.SessionID)
		return
	}
	targetSession := value.(*session.UDPSession)

	if _, err := c.udpListener.WriteToUDPAddrPort(decryptedData, targetSession.GetRemoteAddrPort()); err != nil {
		log.Printf("[ERROR] UDP write error: %v", err)
		c.stats.AddError()
		return
	}

	c.stats.AddBytesSent(uint64(len(decryptedData)))
	c.stats.AddPacketsSent(1)
	targetSession.UpdateLastActive()
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
					c.sessionsByID.Delete(sess.GetSessionID())
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

func isClosedError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, net.ErrClosed) ||
		strings.Contains(err.Error(), "use of closed network connection") ||
		strings.Contains(err.Error(), "connection reset by peer")
}

func isTimeoutError(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}
	return false
}
