package nologyorder

import (
	"context"
	"kabsa/internal/httpclient"
	"kabsa/internal/logging"
	"time"
)

type Client struct {
	http   *httpclient.Client
	logger logging.Logger
}

type CreateOrderRequest struct {
	// whatever your .NET API expects
	CustomerID int64  `json:"customerId"`
	Note       string `json:"note,omitempty"`
}

type CreateOrderResponse struct {
	OrderID int64 `json:"orderId"`
}

func New(baseURL string, timeout time.Duration, logger logging.Logger) (*Client, error) {
	httpCli, err := httpclient.New(baseURL, timeout, logger.With("component", "orders_http"))
	if err != nil {
		return nil, err
	}

	return &Client{
		http:   httpCli,
		logger: logger,
	}, nil
}

func (c *Client) CreateOrder(ctx context.Context, req CreateOrderRequest) (CreateOrderResponse, error) {
	var res CreateOrderResponse
	err := c.http.PostJSON(ctx, "/api/orders", req, &res)
	return res, err
}
