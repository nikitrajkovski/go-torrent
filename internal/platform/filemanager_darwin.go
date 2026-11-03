//go:build darwin

package platform

import "os/exec"

func openDirectory(directory string) error {
	return exec.Command("open", directory).Start()
}
