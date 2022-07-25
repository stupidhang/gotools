package rabbitmq

type Exchange struct {
	Name string
	Type ExchangeType
	// Durable represents restored on server restart.
	Durable bool
	// AutoDelete represents exchanges will be deleted when there are no remaining bindings.
	AutoDelete bool
}

type Queue struct {
	Name string
	// Durable represents queues will survive server restarts.
	Durable bool
	// AutoDelete represents queues will be deleted by the server after a short time.
	// when the last consumer is canceled or the last consumer's channel is closed.
	AutoDelete bool
}

type Bind struct {
	Exchange string // ExchangeName
	Queue    string // QueueName
}

type Publish struct {
	Exchange string      // ExchangeName
	Queue    string      // QueueName or BindKey
	Content  interface{} // Content
}

type message struct {
	contentType  string // MIME content type, example: text/plain
	deliveryMode deliveryMode
	priority     priority
	content      []byte
}

// publishingCustom compared to Publishing, can customize contentType, deliveryMode, priority.
type publishCustom struct {
	Exchange string // ExchangeName
	Key      string // BindKey
	message  message
}

type consumeCustom struct {
	queue   string // QueueName
	autoAck bool   // AutoAck
}

type FanoutPublish struct {
	Exchange string      // ExchangeName
	Queue    string      // QueueName
	Content  interface{} // Content
}

type DefaultPublish struct {
	Queue   string      // QueueName
	Content interface{} // Content
}
