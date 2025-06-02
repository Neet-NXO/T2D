package server

import (
	"bytes"
	"log"
	"time"

	"t2d/internal/protocol"
	"t2d/internal/session"

	"github.com/panjf2000/gnet/v2"
	"github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
)

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

	for buf.Len() >= protocol.HeaderSize {
		headerData := buf.Bytes()[:protocol.HeaderSize]
		header := &protocol.PacketHeader{}
		if err := header.Unmarshal(headerData); err != nil {
			log.Printf("[ERROR] Invalid packet header: %v", err)
			buf.Reset()
			return gnet.None
		}

		if err := header.Validate(); err != nil {
			log.Printf("[ERROR] Invalid packet: %v", err)
			buf.Reset()
			return gnet.None
		}

		totalSize := protocol.HeaderSize + int(header.Length)
		if buf.Len() < totalSize {
			break
		}

		buf.Next(protocol.HeaderSize)
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
			sess := v.(*session.BackendSession)
			if sess.GetDownstreamConn() == c {
				sess.SetDownstreamConn(nil)
			}
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

	for buf.Len() >= protocol.HeaderSize {
		headerData := buf.Bytes()[:protocol.HeaderSize]
		header := &protocol.PacketHeader{}
		if err := header.Unmarshal(headerData); err != nil {
			buf.Reset()
			return gnet.None
		}

		if err := header.Validate(); err != nil {
			buf.Reset()
			return gnet.None
		}

		totalSize := protocol.HeaderSize + int(header.Length)
		if buf.Len() < totalSize {
			break
		}

		buf.Next(protocol.HeaderSize)
		packetData := make([]byte, header.Length)
		buf.Read(packetData)

		// 处理心跳等控制消息
		if header.Type == protocol.PacketTypeHeartbeat {
			// 可以更新连接活跃时间等
			log.Printf("[DEBUG] Received heartbeat from downstream connection")
		}

		// 将下行连接与会话关联
		h.server.connMap.Store(c, header.SessionID)
		if v, ok := h.server.sessions.Load(header.SessionID); ok {
			sess := v.(*session.BackendSession)
			sess.SetDownstreamConn(c)
		}
	}

	return gnet.None
}

func (h *DownstreamHandler) OnTick() (delay time.Duration, action gnet.Action) {
	return time.Second, gnet.None
}
