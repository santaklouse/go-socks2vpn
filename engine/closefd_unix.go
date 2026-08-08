//go:build unix

package engine

import "golang.org/x/sys/unix"

func closeFD(fd int) error {
	return unix.Close(fd)
}
