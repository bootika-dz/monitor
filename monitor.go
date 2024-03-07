// Package monitor implements monitoring of multiple services and calls and
// initiates shutdown of all the services should any of them exit
package monitor

import (
	"context"
	"log"
	"sync"
)

// Service is a generic monitored service
type Service interface {
	// Done returns a channel that is closed when service exits
	Done() <-chan struct{}
}

// Monitor checks and reacts on the change of the status of multiple services
type Monitor struct {
	cancel func()
	mu     sync.Mutex
	cnt    int32
	closed bool
	done   chan struct{}
}

// New returns a new Monitor and a context that monitored services should use
// so they could be cancelled by the monitor.
func New(ctx context.Context) (*Monitor, context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	return &Monitor{
		cancel: cancel,
		done:   make(chan struct{}),
	}, ctx
}

// Done returns a channel that is closed when all the services have exited
func (m *Monitor) Done() <-chan struct{} {
	return m.done
}

// Watch starts monitoring the specified service
func (m *Monitor) Watch(ctx context.Context, name string, srv Service) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		// we're done already
		return
	}
	m.cnt++
	go func() {
		<-srv.Done()
		log.Printf("%s exited", name)
		m.mu.Lock()
		defer m.mu.Unlock()
		m.cancel()
		m.cnt--
		if m.cnt == 0 && !m.closed {
			close(m.done)
			m.closed = true
		}
	}()
}

// Cancel cancels monitoring context and so requests services to exit
func (m *Monitor) Cancel() {
	m.mu.Lock()
	defer m.mu.Unlock()
	// if there are no services being monitored, mark monitor as closed
	if m.cnt == 0 && !m.closed {
		close(m.done)
		m.closed = true
	}
	m.cancel()
}
