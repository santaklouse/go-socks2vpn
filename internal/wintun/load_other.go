//go:build !windows

package wintun

func Load(string) error { return nil }
