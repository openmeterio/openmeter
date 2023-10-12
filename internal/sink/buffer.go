package sink

import (
	"sync"
)

type SinkBuffer struct {
	mu   sync.Mutex
	data map[string]SinkMessage
}

func NewSinkBuffer() SinkBuffer {
	return SinkBuffer{
		data: map[string]SinkMessage{},
	}
}

func (b *SinkBuffer) Size() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.data)
}

func (b *SinkBuffer) Add(message SinkMessage) {
	b.mu.Lock()
	defer b.mu.Unlock()
	// Unique identifier for each message (topic + partition + offset)
	key := message.KafkaMessage.String()
	b.data[key] = message
}

func (b *SinkBuffer) Dequeue() []SinkMessage {
	b.mu.Lock()
	defer b.mu.Unlock()
	list := []SinkMessage{}
	for key, message := range b.data {
		list = append(list, message)
		delete(b.data, key)
	}
	return list
}
