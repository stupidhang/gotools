package amqp

import (
	"errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"time"
)

var (
	ErrConnectionOpenFail = errors.New("amqp connection is not open")

	ErrConnectionRetryFail = errors.New("amqp connection retry failed")
)

const (
	retryNumber = 50

	retryDuration = 3 * time.Second
)

// A socket to connect client with server.
type Connection interface {
	GetConnection() *amqp.Connection
	Reconnect() error
	Close() error
}
