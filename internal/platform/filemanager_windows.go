//go:build windows

package platform

import "os/exec"

func openDirectory(directory string) error {
	return exec.Command("explorer.exe", directory).Start()
}
