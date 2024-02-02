// Package wsman implements a simple WSMAN client interface.
// It assumes you are talking to WSMAN over http(s) and using
// basic authentication.
package wsman

/*
Copyright 2015 Victor Lowther <victor.lowther@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/VictorLowther/simplexml/dom"
	"github.com/VictorLowther/soap"
	"github.com/go-logr/logr"
)

// Client is a thin wrapper around http.Client.
type Client struct {
	OptimizeEnum bool
	Logger       logr.Logger

	httpClient *http.Client
	target     string
	targetPath string
	username   string
	password   string
	useDigest  bool
	challenge  *challenge
}

// Option for setting optional Client values.
type Option func(*Client)

func WithLogger(logger logr.Logger) Option {
	return func(c *Client) {
		c.Logger = logger
	}
}

func WithUseDigest(useDigest bool) Option {
	return func(c *Client) {
		c.useDigest = useDigest
	}
}

// NewClient creates a new wsman.Client.
//
// target must be a URL, and username and password must be the
// username and password to authenticate to the controller with.  If
// username or password are empty, we will not try to authenticate.
// If useDigest is true, we will try to use digest auth instead of
// basic auth.
func NewClient(host *url.URL, username, password string, opts ...Option) *Client {
	defaultClient := &Client{
		httpClient: &http.Client{},
		Logger:     logr.Discard(),
		useDigest:  true,
	}
	for _, opt := range opts {
		opt(defaultClient)
	}

	defaultClient.target = host.String()
	defaultClient.targetPath = host.Path
	defaultClient.username = username
	defaultClient.password = password
	defaultClient.httpClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // not handling certs right now

	}

	return defaultClient
}

func (c *Client) Open(ctx context.Context) error {
	if c.useDigest {
		c.challenge = &challenge{Username: c.username, Password: c.password}
		req, err := http.NewRequestWithContext(ctx, "POST", c.target, nil)
		if err != nil {
			return fmt.Errorf("unable to create request digest auth with %s: %v", c.target, err)
		}
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("unable to perform digest auth with %s: %v", c.target, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 401 {
			return fmt.Errorf("no digest auth at %s", c.target)
		}
		if err := c.challenge.parseChallenge(resp.Header.Get("WWW-Authenticate")); err != nil {
			return fmt.Errorf("failed to parse auth header %v", err)
		}
	}

	return nil
}

func (c *Client) Close() error {
	c.challenge = nil
	return nil
}

// Post overrides http.Client's Post method and adds digest auth handling
// and SOAP pre and post processing.
func (c *Client) Post(ctx context.Context, msg *soap.Message) (response *soap.Message, err error) {
	if c.challenge == nil {
		if err := c.Open(ctx); err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.target, msg.Reader())
	if err != nil {
		return nil, err
	}
	if c.username != "" && c.password != "" {
		if c.useDigest {
			auth, err := c.challenge.authorize("POST", c.targetPath)
			if err != nil {
				return nil, fmt.Errorf("failed digest auth %v", err)
			}
			req.Header.Set("Authorization", auth)
		} else {
			req.SetBasicAuth(c.username, c.password)
		}
	}
	req.Header.Add("content-type", soap.ContentType)
	c.Logger.V(1).Info("debug", "request", req, "body", msg.String())

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if c.useDigest && res.StatusCode == 401 {
		c.Logger.V(1).Info("Digest reauthorizing")
		if err := c.challenge.parseChallenge(res.Header.Get("WWW-Authenticate")); err != nil {
			return nil, err
		}
		auth, err := c.challenge.authorize("POST", c.targetPath)
		if err != nil {
			return nil, fmt.Errorf("failed digest auth %v", err)
		}
		req, err = http.NewRequestWithContext(ctx, "POST", c.target, msg.Reader())
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", auth)
		req.Header.Add("content-type", soap.ContentType)
		res, err = c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
	}

	if res.StatusCode >= 400 {
		b, _ := ioutil.ReadAll(res.Body)
		return nil, fmt.Errorf("wsman.Client: post received %v\n'%v'", res.Status, string(b))
	}
	response, err = soap.Parse(res.Body)
	if err != nil {
		return nil, err
	}

	c.Logger.V(1).Info("debug", "body", response.String())

	return response, nil
}

// Identify performs a basic WSMAN IDENTIFY call.
// The response will provide the version of WSMAN the endpoint
// speaks, along with some details about the WSMAN endpoint itself.
// Note that identify uses soap.Message directly instead of wsman.Message.
func (c *Client) Identify(ctx context.Context) (*soap.Message, error) {
	message := soap.NewMessage()
	message.SetBody(dom.Elem("Identify", NSWSMID))
	return c.Post(ctx, message)
}
