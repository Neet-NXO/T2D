package server

import (
	"log"
	"time"

	"t2d/internal/protocol"
	"t2d/internal/session"
	"t2d/internal/tcpopt"

	"github.com/panjf2000/gnet/v2"
)

type downstreamConnContext struct {
	laneID int
}

// UpstreamHandler 服务端上行处理器
type UpstreamHandler struct {
	server *Server
}

// DownstreamHandler 服务端下行处理器
type DownstreamHandler struct {
	server *Server
}

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
	h.server.applyConnOptions(c, tcpopt.DirectionUpstream)
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
	for c.InboundBuffered() >= protocol.HeaderSize {
		headerData, err := c.Peek(protocol.HeaderSize)
		if err != nil {
			return gnet.None
		}
		header := protocol.PacketHeader{}
		if err := header.Unmarshal(headerData); err != nil {
			log.Printf("[ERROR] Invalid packet header: %v", err)
			c.Discard(c.InboundBuffered())
			return gnet.None
		}

		if err := header.Validate(); err != nil {
			log.Printf("[ERROR] Invalid packet: %v", err)
			c.Discard(c.InboundBuffered())
			return gnet.None
		}

		if int(header.Length) > h.server.maxUpstreamFramePayload() {
			log.Printf("[ERROR] Upstream packet too large: %d", header.Length)
			c.Discard(c.InboundBuffered())
			return gnet.None
		}

		totalSize := protocol.HeaderSize + int(header.Length)
		if c.InboundBuffered() < totalSize {
			break
		}

		if _, err := c.Discard(protocol.HeaderSize); err != nil {
			return gnet.None
		}
		payload, err := c.Next(int(header.Length))
		if err != nil {
			return gnet.None
		}
		h.server.handleUpstreamPacket(&header, payload)
	}

	return gnet.None
}

func (h *UpstreamHandler) OnTick() (delay time.Duration, action gnet.Action) {
	return time.Second, gnet.None
}

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
	h.server.applyConnOptions(c, tcpopt.DirectionDownstream)
	c.SetContext(&downstreamConnContext{laneID: -1})
	return nil, gnet.None
}

func (h *DownstreamHandler) OnClose(c gnet.Conn, err error) gnet.Action {
	if ctx, ok := c.Context().(*downstreamConnContext); ok && ctx.laneID >= 0 {
		h.server.clearDownstreamConnForLane(ctx.laneID, c)
	}

	if err != nil {
		log.Printf("[ERROR] Downstream connection closed with error: %s, %v", c.RemoteAddr(), err)
	} else {
		log.Printf("[INFO] Downstream connection closed: %s", c.RemoteAddr())
	}
	return gnet.None
}

func (h *DownstreamHandler) OnTraffic(c gnet.Conn) gnet.Action {
	for c.InboundBuffered() >= protocol.HeaderSize {
		headerData, err := c.Peek(protocol.HeaderSize)
		if err != nil {
			return gnet.None
		}
		header := protocol.PacketHeader{}
		if err := header.Unmarshal(headerData); err != nil {
			c.Discard(c.InboundBuffered())
			return gnet.None
		}

		if err := header.Validate(); err != nil {
			c.Discard(c.InboundBuffered())
			return gnet.None
		}

		if int(header.Length) > h.server.maxDownstreamControlPayload() {
			log.Printf("[ERROR] Downstream control packet too large: %d", header.Length)
			c.Discard(c.InboundBuffered())
			return gnet.None
		}

		totalSize := protocol.HeaderSize + int(header.Length)
		if c.InboundBuffered() < totalSize {
			break
		}

		if _, err := c.Discard(protocol.HeaderSize); err != nil {
			return gnet.None
		}
		payload, err := c.Next(int(header.Length))
		if err != nil {
			return gnet.None
		}

		if header.Type != protocol.PacketTypeControl {
			continue
		}
		ctrl, err := protocol.UnmarshalLaneControl(payload)
		if err != nil {
			log.Printf("[WARN] Invalid lane control packet from %s: %v", c.RemoteAddr(), err)
			continue
		}
		h.handleLaneControl(c, ctrl)
	}

	return gnet.None
}

func (h *DownstreamHandler) handleLaneControl(c gnet.Conn, ctrl *protocol.LaneControlPayload) {
	if ctrl == nil {
		return
	}
	if int(ctrl.LaneCount) != len(h.server.downWriters) {
		log.Printf("[WARN] Lane count mismatch from %s: got=%d want=%d", c.RemoteAddr(), ctrl.LaneCount, len(h.server.downWriters))
		return
	}
	laneID := int(ctrl.LaneID)
	if laneID < 0 || laneID >= len(h.server.downWriters) {
		return
	}

	ctx, _ := c.Context().(*downstreamConnContext)
	if ctx == nil {
		ctx = &downstreamConnContext{laneID: -1}
		c.SetContext(ctx)
	}

	switch ctrl.Subtype {
	case protocol.ControlSubtypeHello:
		ctx.laneID = laneID
		h.server.setDownstreamConnForLane(laneID, c)
		log.Printf("[INFO] Downstream lane %d registered from %s", laneID, c.RemoteAddr())
	case protocol.ControlSubtypeLaneRegister:
		if ctx.laneID >= 0 && ctx.laneID != laneID {
			log.Printf("[WARN] Lane register mismatch from %s: ctx=%d payload=%d", c.RemoteAddr(), ctx.laneID, laneID)
			return
		}
		ctx.laneID = laneID
		h.server.setDownstreamConnForLane(laneID, c)
		for _, sid := range ctrl.SessionIDs {
			h.server.sessionLaneMap.Store(sid, laneID)
			if v, ok := h.server.sessions.Load(sid); ok {
				sess := v.(*session.BackendSession)
				sess.SetDownstreamConn(c)
			}
		}
	case protocol.ControlSubtypeLaneHeartbeat:
		if ctx.laneID < 0 {
			ctx.laneID = laneID
		}
		h.server.setDownstreamConnForLane(laneID, c)
	default:
		log.Printf("[WARN] Unknown control subtype %d from %s", ctrl.Subtype, c.RemoteAddr())
	}
}

func (h *DownstreamHandler) OnTick() (delay time.Duration, action gnet.Action) {
	return time.Second, gnet.None
}
