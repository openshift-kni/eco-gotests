package iamt

import (
	"context"
	"fmt"
	"net/url"

	"github.com/go-logr/logr"
	"github.com/jacobweinstock/iamt/internal"
	"github.com/jacobweinstock/iamt/internal/wsman"
)

// Client used to perform actions on the machine.
type Client struct {
	Host   string
	Logger logr.Logger
	Pass   string
	Path   string
	Port   uint32
	Scheme string
	User   string

	conn internal.Client
}

// Option for setting optional Client values.
type Option func(*Client)

func WithScheme(scheme string) Option {
	return func(c *Client) {
		c.Scheme = scheme
	}
}

func WithLogger(logger logr.Logger) Option {
	return func(c *Client) {
		c.Logger = logger
	}
}

func WithPort(port uint32) Option {
	return func(c *Client) {
		c.Port = port
	}
}

func WithPath(path string) Option {
	return func(c *Client) {
		c.Path = path
	}
}

// NewClient creates an amt client to use.
func NewClient(host, user, passwd string, opts ...Option) *Client {
	defaultClient := &Client{
		Logger: logr.Discard(),
		Path:   "/wsman",
		Port:   16992,
		Scheme: "http",
	}

	for _, opt := range opts {
		opt(defaultClient)
	}

	defaultClient.Host = host
	defaultClient.User = user
	defaultClient.Pass = passwd
	defaultClient.conn = internal.Client{Log: defaultClient.Logger}

	target := &url.URL{Scheme: defaultClient.Scheme, Host: host + ":" + fmt.Sprint(defaultClient.Port), Path: defaultClient.Path}
	defaultClient.conn.WsmanClient = wsman.NewClient(target, user, passwd)

	return defaultClient
}

// Open the client.
func (c *Client) Open(ctx context.Context) error {
	return c.conn.Open(ctx)
}

// Close the client.
func (c *Client) Close(ctx context.Context) error {
	return c.conn.Close(ctx)
}

// PowerOn will power on a given machine.
func (c *Client) PowerOn(ctx context.Context) error {
	return c.conn.PowerOn(ctx)
}

// PowerOff will power off a given machine.
func (c *Client) PowerOff(ctx context.Context) error {
	return c.conn.PowerOff(ctx)
}

// PowerCycle will power cycle a given machine.
func (c *Client) PowerCycle(ctx context.Context) error {
	return c.conn.PowerCycle(ctx)
}

// SetPXE makes sure the node will pxe boot next time.
func (c *Client) SetPXE(ctx context.Context) error {
	return c.conn.SetPXE(ctx)
}

// IsPoweredOn checks current power state.
func (c *Client) IsPoweredOn(ctx context.Context) (bool, error) {
	return c.conn.IsPoweredOn(ctx)
}
