package broker

import (
	"log/slog"
	"sync/atomic"
)

// Broker fans out room events to connected WebSocket clients.
// LocalBroker delivers within a single process.
// Replace with a Redis-backed implementation for horizontal scaling.
type Broker interface {
	Broadcast(roomID string, payload []byte)
	DroppedCount() int64
	Close()
}

type localMsg struct {
	roomID  string
	payload []byte
}

// LocalBroker is the default single-process broker.
// deliver is called for each message with roomID and the JSON payload.
type LocalBroker struct {
	ch      chan *localMsg
	deliver func(roomID string, payload []byte)
	done    chan struct{}
	dropped atomic.Int64
}

// NewLocalBroker creates a new LocalBroker and starts its dispatch goroutine.
func NewLocalBroker(deliver func(roomID string, payload []byte)) *LocalBroker {
	b := &LocalBroker{
		ch:      make(chan *localMsg, 1000),
		deliver: deliver,
		done:    make(chan struct{}),
	}
	go b.run()
	return b
}

// Broadcast enqueues a message for delivery. If the channel is full the message
// is dropped and the drop counter is incremented.
func (b *LocalBroker) Broadcast(roomID string, payload []byte) {
	select {
	case b.ch <- &localMsg{roomID, payload}:
	default:
		dropped := b.dropped.Add(1)
		slog.Error("broker: channel full, message dropped",
			"room_id", roomID,
			"dropped_total", dropped)
	}
}

// DroppedCount returns the total number of dropped messages since startup.
func (b *LocalBroker) DroppedCount() int64 {
	return b.dropped.Load()
}

func (b *LocalBroker) run() {
	for {
		select {
		case m := <-b.ch:
			b.deliver(m.roomID, m.payload)
		case <-b.done:
			return
		}
	}
}

// Close stops the broker's dispatch goroutine.
func (b *LocalBroker) Close() {
	close(b.done)
}
