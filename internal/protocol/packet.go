package protocol

import (
	"encoding/binary"
	"fmt"
)

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
	MagicNumber = 0x54435050 // "TCPP"
	Version     = 1
	HeaderSize  = 20

	// 包类型
	PacketTypeData      = 1
	PacketTypeHeartbeat = 2
	PacketTypeControl   = 3
)

// Marshal 序列化包头
func (h *PacketHeader) Marshal() []byte {
	buf := make([]byte, HeaderSize)
	h.MarshalTo(buf)
	return buf
}

// MarshalTo 将包头序列化到已有缓冲区
func (h *PacketHeader) MarshalTo(buf []byte) {
	binary.BigEndian.PutUint32(buf[0:4], h.Magic)
	buf[4] = h.Version
	buf[5] = h.Type
	binary.BigEndian.PutUint16(buf[6:8], h.Flags)
	binary.BigEndian.PutUint32(buf[8:12], h.SessionID)
	binary.BigEndian.PutUint32(buf[12:16], h.SequenceID)
	binary.BigEndian.PutUint32(buf[16:20], h.Length)
}

// AppendTo 将包头追加到目标缓冲区
func (h *PacketHeader) AppendTo(dst []byte) []byte {
	offset := len(dst)
	dst = append(dst, make([]byte, HeaderSize)...)
	h.MarshalTo(dst[offset:])
	return dst
}

// AppendPacket 将包头和负载追加到目标缓冲区
func AppendPacket(dst []byte, header *PacketHeader, payload []byte) []byte {
	dst = header.AppendTo(dst)
	return append(dst, payload...)
}

// AppendDataPacket 将数据包追加到目标缓冲区
func AppendDataPacket(dst []byte, sessionID, sequenceID uint32, payload []byte) []byte {
	header := NewDataPacket(sessionID, sequenceID, uint32(len(payload)))
	return AppendPacket(dst, header, payload)
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

// Validate 验证包头
func (h *PacketHeader) Validate() error {
	if h.Magic != MagicNumber {
		return fmt.Errorf("invalid magic number: %x", h.Magic)
	}
	if h.Version != Version {
		return fmt.Errorf("unsupported version: %d", h.Version)
	}
	if h.Type < PacketTypeData || h.Type > PacketTypeControl {
		return fmt.Errorf("invalid packet type: %d", h.Type)
	}
	return nil
}

// NewDataPacket 创建数据包头
func NewDataPacket(sessionID, sequenceID uint32, dataLen uint32) *PacketHeader {
	return &PacketHeader{
		Magic:      MagicNumber,
		Version:    Version,
		Type:       PacketTypeData,
		Flags:      0,
		SessionID:  sessionID,
		SequenceID: sequenceID,
		Length:     dataLen,
	}
}

// NewHeartbeatPacket 创建心跳包头
func NewHeartbeatPacket(sessionID uint32) *PacketHeader {
	return &PacketHeader{
		Magic:      MagicNumber,
		Version:    Version,
		Type:       PacketTypeHeartbeat,
		Flags:      0,
		SessionID:  sessionID,
		SequenceID: 0,
		Length:     0,
	}
}

// NewControlPacket 创建控制包头
func NewControlPacket(sessionID uint32, dataLen uint32) *PacketHeader {
	return &PacketHeader{
		Magic:      MagicNumber,
		Version:    Version,
		Type:       PacketTypeControl,
		Flags:      0,
		SessionID:  sessionID,
		SequenceID: 0,
		Length:     dataLen,
	}
}
