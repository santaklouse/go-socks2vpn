//go:build windows

package elevated

import "golang.org/x/sys/windows"

func IsAdministrator() bool {
	token := windows.GetCurrentProcessToken()
	return token.IsElevated()
}
