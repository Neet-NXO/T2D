package client

import (
	"net"
	"strconv"
	"testing"

	"t2d/internal/session"
)

func BenchmarkSessionLookupRange(b *testing.B) {
	c := &Client{}
	for i := uint32(1); i <= 4096; i++ {
		addr := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: int(10000 + i)}
		c.sessions.Store(addr.String(), session.NewUDPSession(i, nil, addr))
	}
	targetID := uint32(4096)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var found *session.UDPSession
		c.sessions.Range(func(key, value interface{}) bool {
			sess := value.(*session.UDPSession)
			if sess.GetSessionID() == targetID {
				found = sess
				return false
			}
			return true
		})
		if found == nil {
			b.Fatal("session not found")
		}
	}
}

func BenchmarkSessionLookupByID(b *testing.B) {
	c := &Client{}
	for i := uint32(1); i <= 4096; i++ {
		addr := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: int(10000 + i)}
		sess := session.NewUDPSession(i, nil, addr)
		c.sessions.Store(strconv.Itoa(int(i)), sess)
		c.sessionsByID.Store(i, sess)
	}
	targetID := uint32(4096)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		value, ok := c.sessionsByID.Load(targetID)
		if !ok || value == nil {
			b.Fatal("session not found")
		}
	}
}
