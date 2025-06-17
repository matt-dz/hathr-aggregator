package httpclient

import (
	"github.com/hashicorp/go-retryablehttp"
)

type Client struct {
	*retryablehttp.Client
}

func New() *Client {
	client := retryablehttp.NewClient()
	client.RetryMax = 5
	return &Client{
		Client: client,
	}
}
