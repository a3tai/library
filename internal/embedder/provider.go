package embedder

import "sync/atomic"

// Provider is a hot-swappable holder for a *Client. The settings panel
// rebuilds the client when the user saves new LM Studio details; consumers
// (LibraryService search path, MCP server) call Get() each time they need
// the current client. The atomic pointer means swaps and reads don't need a
// mutex on the hot path.
//
// A nil Provider returns a nil *Client — call sites should already handle
// that case (it matches the "embedder not configured" branch that exists
// pre-Provider).
type Provider struct {
	p atomic.Pointer[Client]
}

// NewProvider returns a Provider pre-loaded with `c` (may be nil).
func NewProvider(c *Client) *Provider {
	p := &Provider{}
	p.Set(c)
	return p
}

// Get returns the current client, or nil if none has been set.
func (p *Provider) Get() *Client {
	if p == nil {
		return nil
	}
	return p.p.Load()
}

// Set swaps in a new client atomically. Pass nil to disable embeddings
// entirely (e.g. after the user clears their settings).
func (p *Provider) Set(c *Client) {
	if p == nil {
		return
	}
	p.p.Store(c)
}
