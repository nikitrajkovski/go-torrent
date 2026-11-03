//go:build linux

package platform

import "os/exec"

func openDirectory(directory string) error {
	return exec.Command("xdg-open", directory).Start()
}
