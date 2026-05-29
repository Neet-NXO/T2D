package protocol

import "testing"

func TestStreamBufferHandlesPartialAndCoalescedFrames(t *testing.T) {
	first := AppendDataPacket(nil, 1, 1, []byte("first"))
	second := AppendDataPacket(nil, 2, 1, []byte("second"))
	stream := append(first, second...)

	parser := NewStreamBuffer(64, 1024)
	parser.Write(stream[:10])
	if _, ok, err := parser.Next(); err != nil || ok {
		t.Fatalf("partial frame ok=%v err=%v", ok, err)
	}

	parser.Write(stream[10:])
	frame, ok, err := parser.Next()
	if err != nil || !ok {
		t.Fatalf("first frame ok=%v err=%v", ok, err)
	}
	if frame.Header.SessionID != 1 || string(frame.Payload) != "first" {
		t.Fatalf("unexpected first frame: %+v payload=%q", frame.Header, frame.Payload)
	}

	frame, ok, err = parser.Next()
	if err != nil || !ok {
		t.Fatalf("second frame ok=%v err=%v", ok, err)
	}
	if frame.Header.SessionID != 2 || string(frame.Payload) != "second" {
		t.Fatalf("unexpected second frame: %+v payload=%q", frame.Header, frame.Payload)
	}
	if parser.Len() != 0 {
		t.Fatalf("parser retained %d bytes", parser.Len())
	}
}

func TestStreamBufferRejectsOversizedFrame(t *testing.T) {
	packet := AppendDataPacket(nil, 1, 1, []byte("too-large"))
	parser := NewStreamBuffer(64, 4)
	parser.Write(packet)
	if _, _, err := parser.Next(); err == nil {
		t.Fatalf("expected oversized frame error")
	}
}

func BenchmarkStreamBufferNext(b *testing.B) {
	packet := AppendDataPacket(nil, 1, 1, make([]byte, 1420))
	parser := NewStreamBuffer(len(packet), 2048)
	b.SetBytes(int64(len(packet)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		parser.Write(packet)
		if _, ok, err := parser.Next(); err != nil || !ok {
			b.Fatalf("ok=%v err=%v", ok, err)
		}
	}
}

func BenchmarkStreamBufferNextRef(b *testing.B) {
	packet := AppendDataPacket(nil, 1, 1, make([]byte, 1420))
	parser := NewStreamBuffer(len(packet), 2048)
	b.SetBytes(int64(len(packet)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		parser.Write(packet)
		if _, ok, err := parser.NextRef(); err != nil || !ok {
			b.Fatalf("ok=%v err=%v", ok, err)
		}
	}
}
