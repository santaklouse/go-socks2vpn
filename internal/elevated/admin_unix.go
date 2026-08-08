//go:build !windows

package elevated

import "os"

func IsAdministrator() bool {
	return os.Geteuid() == 0
}
