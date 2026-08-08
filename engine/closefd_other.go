//go:build !unix

package engine

import "errors"

func closeFD(int) error {
	return errors.New("fd-based TUN devices are not supported on this platform")
}
