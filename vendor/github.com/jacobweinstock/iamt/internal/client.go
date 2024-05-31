package internal

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/jacobweinstock/iamt/internal/wsman"
)

type Client struct {
	Log         logr.Logger
	WsmanClient *wsman.Client
}

func (c *Client) Open(ctx context.Context) error {
	return c.WsmanClient.Open(ctx)
}

func (c *Client) Close(_ context.Context) error {
	return c.WsmanClient.Close()
}
