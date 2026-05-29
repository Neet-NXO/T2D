package protocol

import (
	"bytes"
	"testing"
)

func TestPacketHeaderMarshalRoundTrip(t *testing.T) {
	header := NewDataPacket(42, 7, 1400)
	var buf [HeaderSize]byte
	header.MarshalTo(buf[:])

	var decoded PacketHeader
	if err := decoded.Unmarshal(buf[:]); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := decoded.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if decoded != *header {
		t.Fatalf("decoded header mismatch: got %+v want %+v", decoded, *header)
	}
}

func TestAppendDataPacket(t *testing.T) {
	payload := []byte("wireguard-packet")
	packet := AppendDataPacket(nil, 100, 200, payload)
	if len(packet) != HeaderSize+len(payload) {
		t.Fatalf("packet length=%d want=%d", len(packet), HeaderSize+len(payload))
	}

	var decoded PacketHeader
	if err := decoded.Unmarshal(packet[:HeaderSize]); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.SessionID != 100 || decoded.SequenceID != 200 || decoded.Length != uint32(len(payload)) {
		t.Fatalf("unexpected header: %+v", decoded)
	}
	if !bytes.Equal(packet[HeaderSize:], payload) {
		t.Fatalf("payload mismatch")
	}
}

func TestPacketHeaderValidateRejectsInvalid(t *testing.T) {
	header := NewDataPacket(1, 1, 1)
	header.Magic = 0
	if err := header.Validate(); err == nil {
		t.Fatalf("expected invalid magic error")
	}

	header = NewDataPacket(1, 1, 1)
	header.Type = 99
	if err := header.Validate(); err == nil {
		t.Fatalf("expected invalid packet type error")
	}
}

func BenchmarkPacketHeaderMarshal(b *testing.B) {
	header := NewDataPacket(1, 2, 1280)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = header.Marshal()
	}
}

func BenchmarkPacketHeaderMarshalTo(b *testing.B) {
	header := NewDataPacket(1, 2, 1280)
	var buf [HeaderSize]byte
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		header.MarshalTo(buf[:])
	}
}

func BenchmarkAppendDataPacket(b *testing.B) {
	payload := bytes.Repeat([]byte{0x42}, 1420)
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = AppendDataPacket(nil, 1, uint32(i), payload)
	}
}
