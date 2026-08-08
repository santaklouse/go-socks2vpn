package client

import (
	"context"
	"errors"
	"io"
	"log"
	"reflect"
	"testing"

	"github.com/santaklouse/go-socks2vpn/internal/command"
	"github.com/santaklouse/go-socks2vpn/internal/network"
)

type recordingRunner struct {
	failOn string
	calls  []string
}

func (r *recordingRunner) Output(_ context.Context, spec command.Spec) ([]byte, error) {
	r.calls = append(r.calls, spec.String())
	if spec.Name == r.failOn {
		return nil, errors.New("expected failure")
	}
	return nil, nil
}

func TestApplyRollsBackOnlySuccessfulSteps(t *testing.T) {
	runner := &recordingRunner{failOn: "second"}
	session := &networkSession{runner: runner, log: log.New(io.Discard, "", 0)}
	plan := network.Plan{Steps: []network.Step{
		{Do: command.C("first"), Undo: []command.Spec{command.C("undo-first")}},
		{Do: command.C("second"), Undo: []command.Spec{command.C("undo-second")}},
	}}
	if err := session.apply(context.Background(), plan); err == nil {
		t.Fatal("apply unexpectedly succeeded")
	}
	want := []string{"first", "second", "undo-first"}
	if !reflect.DeepEqual(runner.calls, want) {
		t.Fatalf("calls = %#v, want %#v", runner.calls, want)
	}
}

func TestReadProxy(t *testing.T) {
	proxy, err := ReadProxy(stringReader("proxy.example:1080\n"), io.Discard)
	if err != nil || proxy != "proxy.example:1080" {
		t.Fatalf("ReadProxy() = %q, %v", proxy, err)
	}
}

func TestResolveProxyAddressLiteral(t *testing.T) {
	for _, test := range []struct {
		host string
		want string
	}{
		{host: "192.168.192.100", want: "192.168.192.100"},
		{host: "2001:db8::1", want: "2001:db8::1"},
		{host: "::ffff:192.0.2.10", want: "192.0.2.10"},
	} {
		got, err := resolveProxyAddress(context.Background(), test.host)
		if err != nil || got.String() != test.want {
			t.Fatalf("resolveProxyAddress(%q) = %q, %v; want %q", test.host, got, err, test.want)
		}
	}
}

type stringReader string

func (s stringReader) Read(p []byte) (int, error) {
	if len(s) == 0 {
		return 0, io.EOF
	}
	n := copy(p, string(s))
	return n, io.EOF
}
