package protocol

import "testing"

func TestLaneControlMarshalRoundTrip(t *testing.T) {
	in := &LaneControlPayload{
		Subtype:    ControlSubtypeLaneRegister,
		LaneID:     2,
		LaneCount:  4,
		SessionIDs: []uint32{11, 22, 33},
	}
	buf, err := in.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	out, err := UnmarshalLaneControl(buf)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Subtype != in.Subtype || out.LaneID != in.LaneID || out.LaneCount != in.LaneCount {
		t.Fatalf("basic fields mismatch: in=%+v out=%+v", in, out)
	}
	if len(out.SessionIDs) != len(in.SessionIDs) {
		t.Fatalf("session count mismatch: in=%d out=%d", len(in.SessionIDs), len(out.SessionIDs))
	}
	for i := range out.SessionIDs {
		if out.SessionIDs[i] != in.SessionIDs[i] {
			t.Fatalf("session id mismatch at %d: in=%d out=%d", i, in.SessionIDs[i], out.SessionIDs[i])
		}
	}
}

func TestLaneControlRejectInvalid(t *testing.T) {
	if _, err := (&LaneControlPayload{Subtype: ControlSubtypeHello, LaneID: 1, LaneCount: 1}).Marshal(); err == nil {
		t.Fatalf("expected invalid lane id error")
	}
	if _, err := UnmarshalLaneControl([]byte{1, 0, 0, 1, 0, 1, 0, 0}); err == nil {
		t.Fatalf("expected invalid lane id error")
	}
}
