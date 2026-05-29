package stats

import (
	"sync/atomic"
	"time"
)

// Stats 统计信息
type Stats struct {
	BytesSent       atomic.Uint64
	BytesReceived   atomic.Uint64
	PacketsSent     atomic.Uint64
	PacketsReceived atomic.Uint64
	Errors          atomic.Uint64
	ActiveSessions  atomic.Int32
	startTime       time.Time
}

// New 创建新的统计实例
func New() *Stats {
	return &Stats{
		startTime: time.Now(),
	}
}

// AddBytesSent 增加发送字节数
func (s *Stats) AddBytesSent(bytes uint64) {
	s.BytesSent.Add(bytes)
}

// AddBytesReceived 增加接收字节数
func (s *Stats) AddBytesReceived(bytes uint64) {
	s.BytesReceived.Add(bytes)
}

// AddPacketsSent 增加发送包数
func (s *Stats) AddPacketsSent(packets uint64) {
	s.PacketsSent.Add(packets)
}

// AddPacketsReceived 增加接收包数
func (s *Stats) AddPacketsReceived(packets uint64) {
	s.PacketsReceived.Add(packets)
}

// AddError 增加错误计数
func (s *Stats) AddError() {
	s.Errors.Add(1)
}

// AddActiveSession 增加活跃会话数
func (s *Stats) AddActiveSession() {
	s.ActiveSessions.Add(1)
}

// RemoveActiveSession 减少活跃会话数
func (s *Stats) RemoveActiveSession() {
	s.ActiveSessions.Add(-1)
}

// GetBytesSent 获取发送字节数
func (s *Stats) GetBytesSent() uint64 {
	return s.BytesSent.Load()
}

// GetBytesReceived 获取接收字节数
func (s *Stats) GetBytesReceived() uint64 {
	return s.BytesReceived.Load()
}

// GetPacketsSent 获取发送包数
func (s *Stats) GetPacketsSent() uint64 {
	return s.PacketsSent.Load()
}

// GetPacketsReceived 获取接收包数
func (s *Stats) GetPacketsReceived() uint64 {
	return s.PacketsReceived.Load()
}

// GetErrors 获取错误数
func (s *Stats) GetErrors() uint64 {
	return s.Errors.Load()
}

// GetActiveSessions 获取活跃会话数
func (s *Stats) GetActiveSessions() int32 {
	return s.ActiveSessions.Load()
}

// GetUptime 获取运行时间
func (s *Stats) GetUptime() time.Duration {
	return time.Since(s.startTime)
}

// Reset 重置统计信息
func (s *Stats) Reset() {
	s.BytesSent.Store(0)
	s.BytesReceived.Store(0)
	s.PacketsSent.Store(0)
	s.PacketsReceived.Store(0)
	s.Errors.Store(0)
	s.ActiveSessions.Store(0)
	s.startTime = time.Now()
}

// Summary 获取统计摘要
type Summary struct {
	BytesSent       uint64        `json:"bytes_sent"`
	BytesReceived   uint64        `json:"bytes_received"`
	PacketsSent     uint64        `json:"packets_sent"`
	PacketsReceived uint64        `json:"packets_received"`
	Errors          uint64        `json:"errors"`
	ActiveSessions  int32         `json:"active_sessions"`
	Uptime          time.Duration `json:"uptime"`
}

// GetSummary 获取统计摘要
func (s *Stats) GetSummary() Summary {
	return Summary{
		BytesSent:       s.GetBytesSent(),
		BytesReceived:   s.GetBytesReceived(),
		PacketsSent:     s.GetPacketsSent(),
		PacketsReceived: s.GetPacketsReceived(),
		Errors:          s.GetErrors(),
		ActiveSessions:  s.GetActiveSessions(),
		Uptime:          s.GetUptime(),
	}
}
