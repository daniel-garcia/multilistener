package multilistener

import (
	"fmt"
	"net"
)

// Listener implements a net.Listener interface but multiplexes connections
// from multiple listeners.
type Listener struct {
	listeners []net.Listener
	closing   chan struct{}
	conns     chan acceptResults
}

type acceptResults struct {
	conn net.Conn
	err  error
}

var _ net.Listener = &Listener{}

// New creates an instance of a Listener using the given listeners. You must
// pass at least one listener. The new listener object listens for new connection
// on all the given listeners.
func New(listeners ...net.Listener) (*Listener, error) {

	if len(listeners) == 0 {
		return nil, fmt.Errorf("multilistener requires at least 1 listener")
	}

	n := &Listener{
		listeners: listeners,
		closing:   make(chan struct{}),
		conns:     make(chan acceptResults),
	}
	for i := range n.listeners {
		go n.loop(n.listeners[i])
	}
	return n, nil

}

// Addr returns the address of the first listener the multi-listener is using.
// The address of other listeners are not available.
func (l *Listener) Addr() net.Addr {
	return l.listeners[0].Addr()
}

// Close will close the multi-listener by iterating over it's listeners and calling
// Close() on each one. If an error is encountered, it is returned. If multiple
// errors are encountered they are returned in a MutiError. Close will also shut down
// the background goroutines that are calling Accept() on the underlying listeners.
//
// Calling Close() more than once will cause it to panic.
func (l *Listener) Close() error {
	close(l.closing)
	var errors []error
	for i := range l.listeners {
		err := l.listeners[i].Close()
		if err != nil {
			errors = append(errors, err)
		}
	}
	switch len(errors) {
	case 0:
		return nil
	case 1:
		return errors[0]
	}
	return &MultiError{Errors: errors}
}

// MultiError is a wrapper around a slice of errors that implements the error interface.
type MultiError struct {
	Errors []error
}

// Error concats the Error() messages of the underlying errors
func (m *MultiError) Error() string {
	if len(m.Errors) == 0 {
		return ""
	}
	s := "errors: "
	for _, e := range m.Errors {
		s += e.Error() + ", "
	}
	return s
}

// loop continually accepts connections from the given listener. It forwards the result
// of the .Accept() method to a channel on the listener. When a user of the Listener object
// calls Accept(), it receives a value from that channel. Closing the listener will cause
// this loop to exit.
func (l *Listener) loop(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		r := acceptResults{
			conn: conn,
			err:  err,
		}
		select {
		case l.conns <- r:
		case <-l.closing:
			if r.err == nil {
				r.conn.Close()
			}
			return
		}
	}
}

// Accept will wait for a result from calling Accept from the underlying listeners.
// It will return an error if the multi-listener is closed.
func (l *Listener) Accept() (net.Conn, error) {
	select {
	case acceptResult, ok := <-l.conns:
		if ok {
			return acceptResult.conn, acceptResult.err
		}
		return nil, fmt.Errorf("closed conn channel")
	case <-l.closing:
		return nil, fmt.Errorf("listener is closed")
	}
}
