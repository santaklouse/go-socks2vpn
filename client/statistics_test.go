package client

import (
	"testing"
	"time"

	tunengine "github.com/santaklouse/go-socks2vpn/engine"
)

func TestMakeStatistics(t *testing.T) {
	got := makeStatistics(
		tunengine.Statistics{UploadedBytes: 5_120, DownloadedBytes: 12_288},
		tunengine.Statistics{UploadedBytes: 1_024, DownloadedBytes: 4_096},
		2*time.Second,
		5*time.Second,
	)
	if got.UploadBytesPerSecond != 2_048 || got.DownloadBytesPerSecond != 4_096 {
		t.Fatalf("rates = upload %d, download %d", got.UploadBytesPerSecond, got.DownloadBytesPerSecond)
	}
	if got.SessionDuration != 5*time.Second {
		t.Fatalf("duration = %s", got.SessionDuration)
	}
}

func TestCounterDeltaHandlesReset(t *testing.T) {
	if got := counterDelta(12, 42); got != 12 {
		t.Fatalf("counterDelta after reset = %d, want 12", got)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := map[uint64]string{
		0:             "0 B",
		1_023:         "1023 B",
		1_024:         "1.0 KiB",
		1_572_864:     "1.5 MiB",
		1_073_741_824: "1.0 GiB",
	}
	for value, want := range tests {
		if got := FormatBytes(value); got != want {
			t.Errorf("FormatBytes(%d) = %q, want %q", value, got, want)
		}
	}
}
