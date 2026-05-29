package session

import (
	"net"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/gnet/v2"
)

// UDPSession UDP会话
type UDPSession struct {
	sessionID          uint32
	conn               *net.UDPConn
	remoteAddr         *net.UDPAddr
	remoteAddrPort     netip.AddrPort
	lastActiveUnixNano atomic.Int64
	sequenceID         uint32
}

// NewUDPSession 创建新的UDP会话
func NewUDPSession(sessionID uint32, conn *net.UDPConn, remoteAddr *net.UDPAddr) *UDPSession {
	addrPort := remoteAddr.AddrPort()
	return NewUDPSessionAddrPort(sessionID, conn, addrPort)
}

// NewUDPSessionAddrPort 创建新的UDP会话
func NewUDPSessionAddrPort(sessionID uint32, conn *net.UDPConn, remoteAddrPort netip.AddrPort) *UDPSession {
	s := &UDPSession{
		sessionID:      sessionID,
		conn:           conn,
		remoteAddr:     net.UDPAddrFromAddrPort(remoteAddrPort),
		remoteAddrPort: remoteAddrPort,
		sequenceID:     0,
	}
	s.lastActiveUnixNano.Store(time.Now().UnixNano())
	return s
}

// GetSessionID 获取会话ID
func (s *UDPSession) GetSessionID() uint32 {
	return s.sessionID
}

// GetRemoteAddr 获取远程地址
func (s *UDPSession) GetRemoteAddr() *net.UDPAddr {
	return s.remoteAddr
}

// GetRemoteAddrPort 获取远程地址
func (s *UDPSession) GetRemoteAddrPort() netip.AddrPort {
	return s.remoteAddrPort
}

// UpdateLastActive 更新最后活跃时间
func (s *UDPSession) UpdateLastActive() {
	s.lastActiveUnixNano.Store(time.Now().UnixNano())
}

// GetLastActive 获取最后活跃时间
func (s *UDPSession) GetLastActive() time.Time {
	return time.Unix(0, s.lastActiveUnixNano.Load())
}

// NextSequenceID 获取下一个序列号
func (s *UDPSession) NextSequenceID() uint32 {
	return atomic.AddUint32(&s.sequenceID, 1)
}

// GetSequenceID 获取当前序列号
func (s *UDPSession) GetSequenceID() uint32 {
	return atomic.LoadUint32(&s.sequenceID)
}

// IsExpired 检查会话是否过期
func (s *UDPSession) IsExpired(timeout time.Duration) bool {
	return time.Since(s.GetLastActive()) > timeout
}

// Close 关闭会话
func (s *UDPSession) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

// BackendSession 后端会话
type BackendSession struct {
	mu                 sync.RWMutex
	sessionID          uint32
	conn               *net.UDPConn
	backendAddr        *net.UDPAddr
	lastActiveUnixNano atomic.Int64
	sequenceID         uint32
	downstreamConn     gnet.Conn // 下行连接
}

// NewBackendSession 创建新的后端会话
func NewBackendSession(sessionID uint32, conn *net.UDPConn, backendAddr *net.UDPAddr, downstreamConn gnet.Conn) *BackendSession {
	s := &BackendSession{
		sessionID:      sessionID,
		conn:           conn,
		backendAddr:    backendAddr,
		sequenceID:     0,
		downstreamConn: downstreamConn,
	}
	s.lastActiveUnixNano.Store(time.Now().UnixNano())
	return s
}

// GetSessionID 获取会话ID
func (s *BackendSession) GetSessionID() uint32 {
	return s.sessionID
}

// GetBackendAddr 获取后端地址
func (s *BackendSession) GetBackendAddr() *net.UDPAddr {
	return s.backendAddr
}

// GetDownstreamConn 获取下行连接
func (s *BackendSession) GetDownstreamConn() gnet.Conn {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.downstreamConn
}

// SetDownstreamConn 设置下行连接
func (s *BackendSession) SetDownstreamConn(conn gnet.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.downstreamConn = conn
}

// UpdateLastActive 更新最后活跃时间
func (s *BackendSession) UpdateLastActive() {
	s.lastActiveUnixNano.Store(time.Now().UnixNano())
}

// GetLastActive 获取最后活跃时间
func (s *BackendSession) GetLastActive() time.Time {
	return time.Unix(0, s.lastActiveUnixNano.Load())
}

// NextSequenceID 获取下一个序列号
func (s *BackendSession) NextSequenceID() uint32 {
	return atomic.AddUint32(&s.sequenceID, 1)
}

// GetSequenceID 获取当前序列号
func (s *BackendSession) GetSequenceID() uint32 {
	return atomic.LoadUint32(&s.sequenceID)
}

// IsExpired 检查会话是否过期
func (s *BackendSession) IsExpired(timeout time.Duration) bool {
	return time.Since(s.GetLastActive()) > timeout
}

// GetConn 获取UDP连接
func (s *BackendSession) GetConn() *net.UDPConn {
	return s.conn
}

// Close 关闭会话
func (s *BackendSession) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}
