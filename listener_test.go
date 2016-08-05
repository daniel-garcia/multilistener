package multilistener

import (
	"net"
	"testing"
)

func TestNew(t *testing.T) {

	_, err := New()
	if err == nil {
		t.Fatalf("expected error when creating listener with no underlying listeners")
	}
	l1, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("could not create listener")
	}
	l2, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("could not create listener")
	}

	ml, err := New(l1, l2)
	if err != nil {
		t.Fatalf("expected ok, got %s", err)
	}

	c1, err := net.Dial(l1.Addr().Network(), l1.Addr().String())
	if err != nil {
		t.Fatalf("expected to dial to l1: %s", err)
	}
	if n, err := c1.Write([]byte("a")); n != 1 || err != nil {
		t.Fatalf("expected n = 1, err = nil, got %d, %s", n, err)
	}
	c1_ml, err := ml.Accept()
	if err != nil {
		t.Fatalf("expected to connect to c1: %s", err)
	}
	buf := make([]byte, 100)
	if n, err := c1_ml.Read(buf); n != 1 || err != nil {
		t.Fatalf("expected 1 byte got %d, %s", n, err)
	}
	if buf[0] != 'a' {
		t.Fatalf("expected a, got %c", buf[0])
	}

	if err := ml.Close(); err != nil {
		t.Fatalf("expected no error closing: %s", err)
	}

	if _, err := ml.Accept(); err == nil {
		t.Fatalf("expected error after closing")
	}
}
