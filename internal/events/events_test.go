package events

import (
	"context"
	"testing"
	"time"

	"encoding/json"

	"github.com/stretchr/testify/assert"
)

// Fixtures

func messageFixture() *Message {
	return &Message{
		ID:           "some-uuid",
		Content:      map[string]string{"just": "testing"},
		CreationTime: 123456,
		Channel:      "some-channel",
	}
}

func brokerFixture() *MessageBroker {
	return NewMessageBroker("123")
}

func brokerWithoutQueueBuffer() *MessageBroker {
	return &MessageBroker{
		subscribers:   make(map[*Subscriber]struct{}),
		deliveryQueue: make(chan *Message),
	}
}

// Tests

func TestMessageMarshal(t *testing.T) {
	m := messageFixture()
	j, err := json.Marshal(m)
	assert.Nil(t, err, "Could not marshal the message")
	expected := `{"id":"some-uuid","content":{"just":"testing"},"created":123456,"channel_id":"some-channel"}`
	assert.Equal(t, expected, string(j))
}

func TestMessageBroker_Deliver(t *testing.T) {
	m := messageFixture()
	b := brokerFixture()
	assert.Equal(t, 0, len(b.deliveryQueue), "Initialized with non-empty delivery queue")
	err := b.Deliver(m)
	assert.Nil(t, err, "Got error while delivering a message")
	assert.Equal(t, 1, len(b.deliveryQueue), "Expected delivery queue to grow")
	message := <-b.deliveryQueue
	assert.Equal(t, m, message, "Expected: ", m, " but got: ", message)
}

func TestMessageBroker_Deliver_Timeout(t *testing.T) {
	b := brokerWithoutQueueBuffer()
	err := b.Deliver(messageFixture())
	assert.NotNil(t, err, "Expected a delivery timeout error")
}

func TestMessageBroker_Run(t *testing.T) {
	b := brokerFixture()
	ctx, _ := context.WithTimeout(
		context.Background(), time.Duration(5*time.Second))
	s := make(Subscriber)
	b.Subscribe(&s)
	m := messageFixture()
	err := b.Deliver(m)
	if err != nil {
		assert.Fail(t, "Failed to deliver the message")
	}
	go b.Run(ctx)
	select {
	case received := <-s:
		assert.NotNil(t, received)
	case <-time.After(5 * time.Second):
		assert.Fail(t, "Expected message was not dispatched on time")
	}
}

func TestMessageBroker_Subscribe_Unsubscribe(t *testing.T) {
	b := brokerFixture()
	s := make(Subscriber)
	b.Subscribe(&s)
	if _, ok := b.subscribers[&s]; !ok {
		assert.Fail(t, "Failed to subscribe")
	}
	b.Unsubscribe(&s)
	if _, ok := b.subscribers[&s]; ok {
		assert.Fail(t, "Failed to unsubscribe")
	}
}
