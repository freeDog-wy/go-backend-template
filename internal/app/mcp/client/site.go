package client

import (
	"context"
	"encoding/json"
)

func (c *Client) Health(ctx context.Context) (json.RawMessage, error) {
	live, err := c.getPublic(ctx, "/healthz")
	if err != nil {
		return nil, err
	}
	ready, err := c.getPublic(ctx, "/readyz")
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]json.RawMessage{"live": live, "ready": ready})
}
