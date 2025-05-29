package messaging

import (
	"log"

	"github.com/nats-io/nats.go"
)

type Client struct {
	Conn *nats.Conn
}

func New(url string) (*Client, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	log.Println("Connected to NATS")
	return &Client{Conn: nc}, nil
}
