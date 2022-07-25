package rabbitmq

import (
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
	amqp2 "github.com/stupidhang/gotools/amqp"
	"google.golang.org/protobuf/proto"
)

type client struct {
	conn amqp2.Connection
}

// NewClient create a long-lived connection for client.
// url like "amqp://guest:guest@localhost:5672/"
func NewClient(url string) (Client, error) {
	conn, err := amqp2.NewConnection(url)
	if err != nil {
		return nil, err
	}
	return &client{
		conn: conn,
	}, nil
}

// newChannel create a channel when connection is normal.
// Call conn.Reconnect when it's closed.
func (c *client) newChannel() (*amqp.Channel, error) {
	for {
		channel, err := c.conn.GetConnection().Channel()
		if err == nil {
			return channel, nil
		}
		if err == amqp.ErrClosed {
			err := c.conn.Reconnect()
			if err != nil {
				return nil, err
			}
			continue
		}
		return nil, err
	}
}

/*
CreateExchange create exchange with exchange info.
If the exchange does not already exist, the server will create it.
If the exchange exists, the server verifies that it is of the provided type,
durability and auto-delete flags.
*/
func (c *client) CreateExchange(e *Exchange) error {
	if e.Name == "" {
		return nil
	}

	channel, err := c.newChannel()
	if err != nil {
		return err
	}
	defer channel.Close()
	return channel.ExchangeDeclare(e.Name, string(e.Type), e.Durable,
		e.AutoDelete, false, false, nil)
}

/*
CreateQueue creates a queue if it doesn't already exist, or ensures that an
existing queue matches the same parameters.
Durable and Auto-Deleted queues are unlikely to be useful.
*/
func (c *client) CreateQueue(q *Queue) error {
	if q.Name == "" {
		return ErrQueueNameIsEmpty
	}

	channel, err := c.newChannel()
	if err != nil {
		return err
	}
	defer channel.Close()
	_, err = channel.QueueDeclare(q.Name, q.Durable, q.AutoDelete,
		false, false, nil)
	return err
}

/*
Bind bind queue to exchange by queue name.
Durable queues are only able to be bound to durable exchanges.
Non-Durable queues can only be bound to non-durable exchanges.
*/
func (c *client) Bind(b *Bind) error {
	if b.Queue == "" {
		return ErrQueueNameIsEmpty
	}

	channel, err := c.newChannel()
	if err != nil {
		return err
	}
	defer channel.Close()
	return channel.QueueBind(b.Queue, b.Queue, b.Exchange, false, nil)
}

/*
Publish publishing to special exchange and queue.
IF exchange is empty, publishing is passed to the default exchange
with the routingKey of the queue name.
Res ack which represents the success of publishing information.
Content is considered to be json, except proto.Message.
*/
func (c *client) Publish(msg *Publish) (Ack, error) {
	var (
		content []byte
		err     error
	)

	if protoMsg, ok := msg.Content.(proto.Message); ok {
		content, err = proto.Marshal(protoMsg)
		if err != nil {
			return false, err
		}
	} else {
		content, err = json.Marshal(msg.Content)
		if err != nil {
			return false, err
		}
	}

	return c.publishCustom(&publishCustom{
		Exchange: msg.Exchange,
		Key:      msg.Queue,
		message: message{
			contentType:  "text/plain",
			deliveryMode: persistent,
			priority:     priorityZero,
			content:      content,
		},
	})
}

/*
publishCustom publishing with unmandatory and mediate.
Publishings can be undeliverable when the mandatory flag is true and no queue is
bound that matches the routing key, or when the immediate flag is true and no
consumer on the matched queue is ready to accept the delivery.
*/
func (c *client) publishCustom(msg *publishCustom) (Ack, error) {
	channel, err := c.newChannel()
	if err != nil {
		return false, err
	}
	defer channel.Close()
	// Confirm puts this channel into confirm mode.
	err = channel.Confirm(false)
	if err != nil {
		return false, err
	}
	// Add confirm ack.
	confirmChan := channel.NotifyPublish(make(chan amqp.Confirmation, 1))

	for count := 0; count < 5; count++ {
		// Goroutine will exit until message deliver successful.
		err = channel.Publish(msg.Exchange, msg.Key, false, false,
			amqp.Publishing{
				ContentType:  msg.message.contentType,
				DeliveryMode: uint8(msg.message.deliveryMode),
				Priority:     uint8(msg.message.priority),
				Body:         msg.message.content,
			})
		if err != nil {
			return false, err
		}
		confirm := <-confirmChan
		if confirm.Ack {
			return Ack(confirm.Ack), nil
		}
	}

	return false, nil
}

// Consume get message and ack automatically.
func (c *client) Consume(queue string) ([]byte, error) {
	content, _, err := c.consumeCustom(&consumeCustom{
		queue:   queue,
		autoAck: true,
	})
	if err != nil {
		return nil, err
	}
	return content, nil
}

// ConsumeWithAck request user to ack manually.
// If not, message will be persist.
func (c *client) ConsumeWithAck(queue string) ([]byte, chan<- Ack, error) {
	return c.consumeCustom(&consumeCustom{
		queue:   queue,
		autoAck: false,
	})
}

// consumeCustom allow user to customize ack.
// User must manually go back to ack if autoAck is false.
func (c *client) consumeCustom(consume *consumeCustom) ([]byte, chan<- Ack, error) {
	channel, err := c.newChannel()
	if err != nil {
		return nil, nil, err
	}

	deliveryChan, err := channel.Consume(consume.queue,
		"",
		consume.autoAck,
		// When exclusive is false, the server will fairly distribute
		// deliveries across multiple consumers.
		false,
		// The noLocal flag is not supported by RabbitMQ.
		false,
		// When noWait is true, do not wait for the server to confirm the request and
		// immediately begin deliveries.
		false,
		nil,
	)
	if err != nil {
		return nil, nil, err
	}

	delivery := <-deliveryChan
	var ackChan chan Ack

	if !consume.autoAck {
		ackChan = make(chan Ack, 1)
		go consumeAck(channel, delivery, ackChan)
	} else {
		defer channel.Close()
	}
	return delivery.Body, ackChan, nil
}

// consumeAck to accept user's ack and send to mq.
func consumeAck(channel *amqp.Channel, delivery amqp.Delivery, ackChan <-chan Ack) {
	defer channel.Close()
	ack := <-ackChan
	_ = delivery.Ack(bool(ack))
}

// CreateDefaultQueue create queue with durable, notAutoDelete which bind on default exchange automatically
func (c *client) CreateDefaultQueue(queue string) error {
	return c.CreateQueue(&Queue{
		Name:       queue,
		Durable:    true,
		AutoDelete: false,
	})
}

// PublishFanout create fanout-type exchange and queue then publish content after bind them.
func (c *client) PublishFanout(publish *FanoutPublish) error {
	err := c.createFanoutExchange(publish.Exchange)
	if err != nil {
		return err
	}
	err = c.CreateDefaultQueue(publish.Queue)
	if err != nil {
		return err
	}
	err = c.Bind(&Bind{
		Exchange: publish.Exchange,
		Queue:    publish.Queue,
	})
	if err != nil {
		return err
	}

	_, err = c.Publish(&Publish{
		Exchange: publish.Exchange,
		Content:  publish.Content,
	})
	return err
}

// createFanoutExchange create exchange with fanout, durable, notAutoDelete.
func (c *client) createFanoutExchange(name string) error {
	return c.CreateExchange(&Exchange{
		Name:       name,
		Type:       Fanout,
		Durable:    true,
		AutoDelete: false,
	})
}

// PublishDefault create a queue which is bound to the default exchange,
// then publish to the default exchange.
func (c *client) PublishDefault(publish *DefaultPublish) error {
	err := c.CreateDefaultQueue(publish.Queue)
	if err != nil {
		return err
	}
	_, err = c.Publish(&Publish{
		Exchange: "",
		Queue:    publish.Queue,
		Content:  publish.Content,
	})
	return err
}
