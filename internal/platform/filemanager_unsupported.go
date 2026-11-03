//go:build !linux && !windows && !darwin

package platform

import "fmt"

func openDirectory(_ string) error {
	return fmt.Errorf("opening the file manager is unsupported on this platform")
}
