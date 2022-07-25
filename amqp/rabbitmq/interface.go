package rabbitmq

import "errors"

var (
	ErrQueueNameIsEmpty = errors.New("queue name is empty")
)

/*
Client, simplified application of mq.
All queue bind exchange with queue name, so there is no key param.
*/
type Client interface {
	CreateExchange(exchange *Exchange) error
	CreateQueue(queue *Queue) error
	Bind(bind *Bind) error
	Publish(msg *Publish) (Ack, error)
	Consume(queue string) ([]byte, error)
	ConsumeWithAck(queue string) ([]byte, chan<- Ack, error)

	CreateDefaultQueue(queue string) error
	PublishFanout(publish *FanoutPublish) error
	PublishDefault(publish *DefaultPublish) error
}

type ExchangeType string

//goland:noinspection SpellCheckingInspection
const (
	Direct  ExchangeType = "direct"  // Direct exchange
	Fanout  ExchangeType = "fanout"  // Fanout exchange
	Topic   ExchangeType = "topic"   // Topic exchange
	Headers ExchangeType = "headers" // Headers exchange
)

/*
DeliveryMode. Transient means higher throughput but messages will not be
restored on broker restart.  The delivery mode of publishings is unrelated
to the durability of the queues they reside on.  Transient messages will
not be restored to durable queues, persistent messages will be restored to
durable queues and lost on non-durable queues during server restart.
*/
type deliveryMode uint8

// nolint
const (
	transient deliveryMode = iota + 1 // Transient means higher throughput
	persistent
)

// priority 0 to 9
type priority uint8

// nolint
const (
	priorityZero priority = iota
	priorityOne
	priorityTwo
	priorityThree
	priorityFour
	priorityFive
	prioritySix
	prioritySeven
	priorityEight
	priorityNine
)

// Ack is used to Acknowledge consume message.
type Ack bool

const (
	Acknowledge   Ack = true
	UnAcknowledge Ack = false
)
