package protocol

import "fmt"

// Frame 表示从TCP字节流中解析出的完整协议帧。
type Frame struct {
	Header  PacketHeader
	Payload []byte
}

// StreamBuffer 累积TCP字节流并按T2D包头切帧。
type StreamBuffer struct {
	buf        []byte
	off        int
	maxPayload int
}

// NewStreamBuffer 创建流式分帧缓冲区。
func NewStreamBuffer(capacity, maxPayload int) *StreamBuffer {
	if capacity < HeaderSize {
		capacity = HeaderSize
	}
	return &StreamBuffer{
		buf:        make([]byte, 0, capacity),
		maxPayload: maxPayload,
	}
}

// Write 追加新读取到的TCP字节。
func (b *StreamBuffer) Write(p []byte) {
	if b.off > 0 && len(b.buf)+len(p) > cap(b.buf) {
		copy(b.buf, b.buf[b.off:])
		b.buf = b.buf[:len(b.buf)-b.off]
		b.off = 0
	}
	b.buf = append(b.buf, p...)
}

// Len 返回当前缓存字节数。
func (b *StreamBuffer) Len() int {
	return len(b.buf) - b.off
}

// Reset 清空缓存。
func (b *StreamBuffer) Reset() {
	b.buf = b.buf[:0]
	b.off = 0
}

// Next 返回下一个完整帧；当数据不足时ok=false。返回的Payload是独立副本。
func (b *StreamBuffer) Next() (frame Frame, ok bool, err error) {
	frame, ok, err = b.NextRef()
	if err != nil || !ok {
		return frame, ok, err
	}
	payload := make([]byte, len(frame.Payload))
	copy(payload, frame.Payload)
	frame.Payload = payload
	return frame, true, nil
}

// NextRef 返回下一个完整帧，Payload引用内部缓冲区。
// 调用方必须在下一次Write、Reset或NextRef之前处理完Payload。
func (b *StreamBuffer) NextRef() (frame Frame, ok bool, err error) {
	if b.Len() < HeaderSize {
		return Frame{}, false, nil
	}

	var header PacketHeader
	if err := header.Unmarshal(b.buf[b.off : b.off+HeaderSize]); err != nil {
		return Frame{}, false, err
	}
	if err := header.Validate(); err != nil {
		return Frame{}, false, err
	}
	if b.maxPayload > 0 && int(header.Length) > b.maxPayload {
		return Frame{}, false, fmt.Errorf("packet payload too large: %d > %d", header.Length, b.maxPayload)
	}

	totalLen := HeaderSize + int(header.Length)
	if b.Len() < totalLen {
		return Frame{}, false, nil
	}

	payloadStart := b.off + HeaderSize
	payloadEnd := b.off + totalLen
	payload := b.buf[payloadStart:payloadEnd]
	b.off += totalLen
	if b.off == len(b.buf) {
		b.Reset()
	}

	return Frame{Header: header, Payload: payload}, true, nil
}
