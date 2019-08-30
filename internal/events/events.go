package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"
)

const (
	DeliveryQueueCap   = 1000
	DeliveryTimeoutSec = 5
)

// Message represents a message that can be provided to the event broker
type Message struct {
	ID           string            `json:"id"`
	Content      map[string]string `json:"content"`
	CreationTime int64             `json:"created"`
	Channel      string            `json:"channel_id"`
}

type Subscriber chan []byte

type MessageBroker struct {
	messageId     string
	subscribers   map[*Subscriber]struct{}
	deliveryQueue chan *Message
}

func (b *MessageBroker) Run(ctx context.Context) {
	for {
		select {
		case m := <-b.deliveryQueue:
			for s := range b.subscribers {
				j, err := json.Marshal(m)
				if err != nil {
					log.Print("Could not marshal message: ", m.ID)
				}
				*s <- j
			}
		}
	}
}

func NewMessageBroker(id string) *MessageBroker {
	return &MessageBroker{
		messageId:     id,
		subscribers:   make(map[*Subscriber]struct{}),
		deliveryQueue: make(chan *Message, DeliveryQueueCap),
	}
}

func (b *MessageBroker) Subscribe(c *Subscriber) {
	b.subscribers[c] = struct{}{}
}

func (b *MessageBroker) Unsubscribe(c *Subscriber) {
	delete(b.subscribers, c)
}

// Deliver appends the given message to the delivery queue.
func (b *MessageBroker) Deliver(m *Message) error {
	select {
	case b.deliveryQueue <- m:
		log.Print("Delivered message: ", m.ID)
	case <-time.After(DeliveryTimeoutSec * time.Second):
		errorMessage := fmt.Sprintf("Timed out while trying to deliver a message: %s\n", m.ID)
		log.Print(errorMessage)
		return errors.New(errorMessage)
	}
	return nil
}
