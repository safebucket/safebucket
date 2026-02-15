package messaging

import (
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

const testTimeout = 2 * time.Second

func receiveOne(t *testing.T, ch <-chan *message.Message) *message.Message {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(testTimeout):
		t.Fatal("timed out waiting for message")
		return nil
	}
}

func TestNewMemoryPubSub(t *testing.T) {
	ch := NewMemoryChannel()
	pub := NewMemoryPublisher(ch, "test-topic")
	sub := NewMemorySubscriber(ch, "test-topic")
	if pub == nil {
		t.Fatal("expected non-nil publisher")
	}
	if sub == nil {
		t.Fatal("expected non-nil subscriber")
	}
}

func TestMemoryPublishAndSubscribe(t *testing.T) {
	ch := NewMemoryChannel()
	pub := NewMemoryPublisher(ch, "test-topic")
	sub := NewMemorySubscriber(ch, "test-topic")
	defer pub.Close()

	msgCh := sub.Subscribe()

	uuid := watermill.NewUUID()
	payload := []byte("hello world")
	err := pub.Publish(message.NewMessage(uuid, payload))
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	msg := receiveOne(t, msgCh)
	if msg.UUID != uuid {
		t.Errorf("expected UUID %s, got %s", uuid, msg.UUID)
	}
	if string(msg.Payload) != string(payload) {
		t.Errorf("expected payload %q, got %q", payload, msg.Payload)
	}
	msg.Ack()
}

func TestMemoryPublishMultipleMessages(t *testing.T) {
	ch := NewMemoryChannel()
	pub := NewMemoryPublisher(ch, "test-topic")
	sub := NewMemorySubscriber(ch, "test-topic")
	defer pub.Close()

	msgCh := sub.Subscribe()

	const count = 5
	expected := make(map[string]bool, count)
	for i := range count {
		uuid := watermill.NewUUID()
		expected[uuid] = false
		err := pub.Publish(message.NewMessage(uuid, []byte("msg")))
		if err != nil {
			t.Fatalf("Publish %d failed: %v", i, err)
		}
	}

	for range count {
		msg := receiveOne(t, msgCh)
		if _, ok := expected[msg.UUID]; !ok {
			t.Errorf("received unexpected UUID %s", msg.UUID)
		}
		expected[msg.UUID] = true
		msg.Ack()
	}

	for uuid, received := range expected {
		if !received {
			t.Errorf("message %s was never received", uuid)
		}
	}
}

func TestMemoryMessageAck(t *testing.T) {
	ch := NewMemoryChannel()
	pub := NewMemoryPublisher(ch, "test-topic")
	sub := NewMemorySubscriber(ch, "test-topic")
	defer pub.Close()

	msgCh := sub.Subscribe()

	err := pub.Publish(message.NewMessage(watermill.NewUUID(), []byte("ack-test")))
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	msg := receiveOne(t, msgCh)
	msg.Ack()

	// Verify Ack was accepted by publishing and receiving another message.
	err = pub.Publish(message.NewMessage(watermill.NewUUID(), []byte("after-ack")))
	if err != nil {
		t.Fatalf("Publish after ack failed: %v", err)
	}

	msg2 := receiveOne(t, msgCh)
	if string(msg2.Payload) != "after-ack" {
		t.Errorf("expected payload %q, got %q", "after-ack", msg2.Payload)
	}
	msg2.Ack()
}

func TestMemoryPublisherClose(t *testing.T) {
	ch := NewMemoryChannel()
	pub := NewMemoryPublisher(ch, "test-topic")

	err := pub.Close()
	if err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	err = pub.Publish(message.NewMessage(watermill.NewUUID(), []byte("after-close")))
	if err == nil {
		t.Error("expected error when publishing after Close, got nil")
	}
}

func TestMemorySubscriberClose(t *testing.T) {
	ch := NewMemoryChannel()
	sub := NewMemorySubscriber(ch, "test-topic")

	err := sub.Close()
	if err != nil {
		t.Fatalf("subscriber Close returned error: %v", err)
	}
}

func TestMemoryIndependentTopics(t *testing.T) {
	ch1 := NewMemoryChannel()
	pub1 := NewMemoryPublisher(ch1, "topic-a")
	sub1 := NewMemorySubscriber(ch1, "topic-a")
	defer pub1.Close()
	ch2 := NewMemoryChannel()
	pub2 := NewMemoryPublisher(ch2, "topic-b")
	sub2 := NewMemorySubscriber(ch2, "topic-b")
	defer pub2.Close()

	msgCh1 := sub1.Subscribe()
	msgCh2 := sub2.Subscribe()

	uuid := watermill.NewUUID()
	err := pub1.Publish(message.NewMessage(uuid, []byte("only-topic-a")))
	if err != nil {
		t.Fatalf("Publish to topic-a failed: %v", err)
	}

	// topic-a subscriber should receive the message.
	msg := receiveOne(t, msgCh1)
	if msg.UUID != uuid {
		t.Errorf("expected UUID %s, got %s", uuid, msg.UUID)
	}
	msg.Ack()

	// topic-b subscriber should NOT receive the message.
	select {
	case m := <-msgCh2:
		t.Errorf("topic-b should not have received a message, got UUID %s", m.UUID)
	case <-time.After(200 * time.Millisecond):
		// expected: no message on topic-b
	}
	_ = sub2.Close()
}
