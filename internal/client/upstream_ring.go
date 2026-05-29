package client

import (
	"sync/atomic"
	"time"
)

type upstreamPacket struct {
	buf  []byte
	data []byte
}

type cachePad [64]byte

type upstreamRing struct {
	slots    []*upstreamPacket
	mask     uint64
	capacity uint64
	_        cachePad
	tail     atomic.Uint64
	_        cachePad
	freeHead atomic.Uint64
	_        cachePad
	closed   atomic.Bool
	_        cachePad

	readHead uint64
}

func newUpstreamRing(capacity, packetSize int) *upstreamRing {
	if capacity <= 0 {
		capacity = 1
	}
	capacity = nextPowerOfTwo(capacity)
	r := &upstreamRing{
		slots:    make([]*upstreamPacket, capacity),
		mask:     uint64(capacity - 1),
		capacity: uint64(capacity),
	}
	for i := 0; i < capacity; i++ {
		r.slots[i] = &upstreamPacket{buf: make([]byte, packetSize)}
	}
	return r
}

func (r *upstreamRing) acquire() *upstreamPacket {
	wait := 0
	for {
		if r.closed.Load() {
			return nil
		}
		tail := r.tail.Load()
		if tail-r.freeHead.Load() < r.capacity {
			p := r.slots[tail&r.mask]
			p.data = p.data[:0]
			return p
		}
		ringWait(&wait)
	}
}

func (r *upstreamRing) submit(p *upstreamPacket) bool {
	if p == nil || r.closed.Load() {
		return false
	}
	r.tail.Add(1)
	return true
}

func (r *upstreamRing) pop() *upstreamPacket {
	wait := 0
	for {
		if r.readHead != r.tail.Load() {
			p := r.slots[r.readHead&r.mask]
			r.readHead++
			return p
		}
		if r.closed.Load() {
			return nil
		}
		ringWait(&wait)
	}
}

func (r *upstreamRing) tryPop() *upstreamPacket {
	if r.readHead == r.tail.Load() {
		return nil
	}
	p := r.slots[r.readHead&r.mask]
	r.readHead++
	return p
}

func (r *upstreamRing) release(p *upstreamPacket) {
	if p == nil {
		return
	}
	p.data = p.data[:0]
	r.freeHead.Add(1)
}

func (r *upstreamRing) discard(p *upstreamPacket) {
	if p != nil {
		p.data = p.data[:0]
	}
}

func (r *upstreamRing) close() {
	r.closed.Store(true)
}

func (r *upstreamRing) inflight() int {
	inflight := r.tail.Load() - r.freeHead.Load()
	if inflight > r.capacity {
		inflight = r.capacity
	}
	return int(inflight)
}

func nextPowerOfTwo(n int) int {
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}

func ringWait(count *int) {
	switch {
	case *count < 32:
		*count++
	case *count < 64:
		*count++
	default:
		time.Sleep(time.Millisecond)
	}
}
