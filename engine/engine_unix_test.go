//go:build unix

package engine

import (
	"fmt"
	"testing"

	"golang.org/x/sys/unix"
)

func TestStartOwnsFDOnFailure(t *testing.T) {
	fds := []int{-1, -1}
	if err := unix.Pipe(fds); err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fds[1])

	instance := New()
	err := instance.Start(Config{
		Device:   fmt.Sprintf("fd://%d", fds[0]),
		ProxyURL: "http://127.0.0.1:8080",
		MTU:      1500,
	})
	if err == nil {
		t.Fatal("Start unexpectedly accepted an HTTP proxy")
	}
	if err := unix.Close(fds[0]); err != unix.EBADF {
		t.Fatalf("TUN fd was not closed after Start failure: %v", err)
	}
}

func TestStartAndStopWithFDDevice(t *testing.T) {
	fds := []int{-1, -1}
	if err := unix.Pipe(fds); err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fds[1])

	instance := New()
	if err := instance.Start(Config{
		Device:   fmt.Sprintf("fd://%d", fds[0]),
		ProxyURL: "socks5://127.0.0.1:1080",
		MTU:      1500,
	}); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	instance.Stop()
	if err := unix.Close(fds[0]); err != unix.EBADF {
		t.Fatalf("TUN fd was not closed by Stop: %v", err)
	}
}
