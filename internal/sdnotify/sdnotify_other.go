//go:build !linux

package sdnotify

func SendReloading() string {
	return ""
}
