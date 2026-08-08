package engine

import "testing"

func TestRejectUDPPort(t *testing.T) {
	rejected := map[uint16]struct{}{
		443: {},
	}

	if !rejectUDPPort(443, rejected) {
		t.Fatal("UDP/443 was not rejected")
	}
	if rejectUDPPort(53, rejected) {
		t.Fatal("UDP/53 was unexpectedly rejected")
	}
}
