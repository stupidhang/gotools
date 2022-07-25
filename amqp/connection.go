package amqp

import (
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

type connection struct {
	url  string
	conn *amqp.Connection
	mu   sync.Mutex
}

func NewConnection(url string) (Connection, error) {
	c := connection{
		url: url,
		mu:  sync.Mutex{},
	}
	// create a connection
	err := c.reconnect()
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// reconnect lazy mode to create a connection.
// Create a daemon goroutine to listen connection close.
func (c *connection) reconnect() error {
	if c.conn != nil && !c.conn.IsClosed() {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil && !c.conn.IsClosed() {
		return nil
	}

	var (
		conn *amqp.Connection
		err  error
	)
	for i := 0; i < retryNumber; i++ {
		conn, err = amqp.Dial(c.url)
		if err == nil {
			c.conn = conn
			break
		}
		logrus.Info(ErrConnectionOpenFail)
		<-time.After(retryDuration)
	}
	if err != nil {
		return ErrConnectionRetryFail
	}

	// Daemon goroutine to accept connection close and reconnect.
	go func() {
		ch := make(chan *amqp.Error)
		<-c.conn.NotifyClose(ch)
		_ = c.reconnect()
	}()
	return nil
}

func (c *connection) GetConnection() *amqp.Connection {
	return c.conn
}

func (c *connection) Reconnect() error {
	return c.reconnect()
}
func (c *connection) Close() error {
	return c.conn.Close()
}
