package client

type Option func(*Client) error

// Use specific negotiation style if possible.
func WithNegotiationStyle(style NegotiationStyle) Option {
	return func(c *Client) error {
		c.style = style
		return nil
	}
}

// Return error, if it cannot use specific negotiation style.
func WithForceStyle(b bool) Option {
	return func(c *Client) error {
		c.forceStyle = b
		return nil
	}
}

// Use structured reply in transmission phase.
func WithStructuredReply(b bool) Option {
	return func(c *Client) error {
		c.structuredReply = b
		return nil
	}
}