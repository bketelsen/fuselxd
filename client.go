package main

import (
	client "github.com/lxc/lxd/client"

	"github.com/pkg/errors"
)

type Client struct {
	URL string

	conn client.ContainerServer
}

// NewClient creates a new connection to an LXD Daemon,
// returning a Client
func NewClient(url string) (*Client, error) {
	c := &Client{
		URL: url,
	}
	err := c.Connect()
	return c, err
}

// Connect establishes a connection to an LXD Daemon
func (c *Client) Connect() error {
	var err error
	c.conn, err = client.ConnectLXDUnix(c.URL, nil)
	if err != nil {
		return errors.Wrap(err, "Error connecting to LXD daemon")
	}

	return nil
}
