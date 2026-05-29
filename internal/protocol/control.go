package protocol

import (
	"encoding/binary"
	"fmt"
)

const (
	ControlSubtypeHello         uint8 = 1
	ControlSubtypeLaneRegister  uint8 = 2
	ControlSubtypeLaneHeartbeat uint8 = 3
)

// LaneControlPayload 表示控制帧载荷。
type LaneControlPayload struct {
	Subtype    uint8
	LaneID     uint16
	LaneCount  uint16
	SessionIDs []uint32
}

// Marshal 将控制载荷编码为字节。
func (p *LaneControlPayload) Marshal() ([]byte, error) {
	if p == nil {
		return nil, fmt.Errorf("nil lane control payload")
	}
	if p.LaneCount == 0 {
		return nil, fmt.Errorf("invalid lane count: 0")
	}
	if p.LaneID >= p.LaneCount {
		return nil, fmt.Errorf("invalid lane id: %d >= %d", p.LaneID, p.LaneCount)
	}
	if len(p.SessionIDs) > 0xFFFF {
		return nil, fmt.Errorf("too many session ids: %d", len(p.SessionIDs))
	}

	buf := make([]byte, 8+len(p.SessionIDs)*4)
	buf[0] = p.Subtype
	buf[1] = 0
	binary.BigEndian.PutUint16(buf[2:4], p.LaneID)
	binary.BigEndian.PutUint16(buf[4:6], p.LaneCount)
	binary.BigEndian.PutUint16(buf[6:8], uint16(len(p.SessionIDs)))
	off := 8
	for _, sid := range p.SessionIDs {
		binary.BigEndian.PutUint32(buf[off:off+4], sid)
		off += 4
	}
	return buf, nil
}

// UnmarshalLaneControl 解析控制载荷。
func UnmarshalLaneControl(payload []byte) (*LaneControlPayload, error) {
	if len(payload) < 8 {
		return nil, fmt.Errorf("control payload too short: %d", len(payload))
	}
	sessionCount := int(binary.BigEndian.Uint16(payload[6:8]))
	expected := 8 + sessionCount*4
	if len(payload) != expected {
		return nil, fmt.Errorf("control payload size mismatch: got=%d want=%d", len(payload), expected)
	}

	result := &LaneControlPayload{
		Subtype:   payload[0],
		LaneID:    binary.BigEndian.Uint16(payload[2:4]),
		LaneCount: binary.BigEndian.Uint16(payload[4:6]),
	}
	if result.LaneCount == 0 {
		return nil, fmt.Errorf("invalid lane count: 0")
	}
	if result.LaneID >= result.LaneCount {
		return nil, fmt.Errorf("invalid lane id: %d >= %d", result.LaneID, result.LaneCount)
	}

	if sessionCount > 0 {
		result.SessionIDs = make([]uint32, sessionCount)
		off := 8
		for i := 0; i < sessionCount; i++ {
			result.SessionIDs[i] = binary.BigEndian.Uint32(payload[off : off+4])
			off += 4
		}
	}
	return result, nil
}

// NewControlPacketPayload 创建控制帧头与载荷。
func NewControlPacketPayload(sequenceID uint32, ctrl *LaneControlPayload) (*PacketHeader, []byte, error) {
	payload, err := ctrl.Marshal()
	if err != nil {
		return nil, nil, err
	}
	header := NewControlPacket(uint32(ctrl.LaneID), uint32(len(payload)))
	header.SequenceID = sequenceID
	return header, payload, nil
}
