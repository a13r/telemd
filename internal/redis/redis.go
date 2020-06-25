package redis

import (
	goredis "github.com/go-redis/redis/v7"
	"math"
	"sync"
	"time"
)

type ConnectionState uint8

const (
	Connected ConnectionState = 1 // first connection
	Failed    ConnectionState = 2 // re-connect failed
	Recovered ConnectionState = 3 // successfully re-connected after failure
)

type ReconnectingClient struct {
	Client          *goredis.Client
	ConnectionState chan ConnectionState
	pingMu          sync.Mutex
}

func NewReconnectingClient(options *goredis.Options, retryBackoff time.Duration) *ReconnectingClient {
	connectionState := make(chan ConnectionState)
	client := goredis.NewClient(options)
	options.MaxRetries = math.MaxInt32
	options.Limiter = newLimiter(retryBackoff, connectionState)
	return &ReconnectingClient{
		Client:          client,
		ConnectionState: connectionState,
	}
}

func NewReconnectingClientFromUrl(url string, retryBackoff time.Duration) (*ReconnectingClient, error) {
	options, err := goredis.ParseURL(url)

	if err != nil {
		return nil, err
	}
	return NewReconnectingClient(options, retryBackoff), nil
}

func (c *ReconnectingClient) InitiateConnection() {
	// ensure there is only one ping
	c.pingMu.Lock()
	c.Client.Ping()
	c.pingMu.Unlock()
}

func (c *ReconnectingClient) Close() {
	_ = c.Client.Close()
	close(c.ConnectionState)
}
