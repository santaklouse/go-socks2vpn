package client

import (
	"context"
	"fmt"
	"time"

	tunengine "github.com/santaklouse/go-socks2vpn/engine"
)

const statisticsInterval = time.Second

// Statistics is a point-in-time view of traffic carried by one VPN session.
type Statistics struct {
	UploadedBytes          uint64
	DownloadedBytes        uint64
	UploadBytesPerSecond   uint64
	DownloadBytesPerSecond uint64
	SessionDuration        time.Duration
}

func watchStatistics(ctx context.Context, read func() tunengine.Statistics, report func(Statistics), interval time.Duration) {
	startedAt := time.Now()
	previousAt := startedAt
	previous := read()
	report(makeStatistics(previous, previous, 0, 0))

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case now := <-ticker.C:
			current := read()
			report(makeStatistics(current, previous, now.Sub(previousAt), now.Sub(startedAt)))
			previous = current
			previousAt = now
		case <-ctx.Done():
			now := time.Now()
			current := read()
			report(makeStatistics(current, previous, now.Sub(previousAt), now.Sub(startedAt)))
			return
		}
	}
}

func makeStatistics(current, previous tunengine.Statistics, sampleDuration, sessionDuration time.Duration) Statistics {
	result := Statistics{
		UploadedBytes:   current.UploadedBytes,
		DownloadedBytes: current.DownloadedBytes,
		SessionDuration: sessionDuration,
	}
	if sampleDuration <= 0 {
		return result
	}
	seconds := sampleDuration.Seconds()
	result.UploadBytesPerSecond = uint64(float64(counterDelta(current.UploadedBytes, previous.UploadedBytes)) / seconds)
	result.DownloadBytesPerSecond = uint64(float64(counterDelta(current.DownloadedBytes, previous.DownloadedBytes)) / seconds)
	return result
}

func counterDelta(current, previous uint64) uint64 {
	if current < previous {
		return current
	}
	return current - previous
}

// FormatBytes formats a byte count using binary units suitable for live UI.
func FormatBytes(value uint64) string {
	const unit = uint64(1024)
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}
	divisor := unit
	unitIndex := 0
	for quotient := value / unit; quotient >= unit && unitIndex < 4; quotient /= unit {
		divisor *= unit
		unitIndex++
	}
	return fmt.Sprintf("%.1f %ciB", float64(value)/float64(divisor), "KMGTPE"[unitIndex])
}

// FormatRate formats a byte-per-second value for CLI and GUI displays.
func FormatRate(value uint64) string {
	return FormatBytes(value) + "/s"
}
