//go:build !windows

package watch

import (
	"os"

	"golang.org/x/sys/unix"
)

// lockFile takes an exclusive advisory lock on path, waiting for it. The
// kernel releases it when the holder exits, so a crashed process never
// leaves the store locked.
func lockFile(path string) (unlock func(), err error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, err
	}
	return func() {
		_ = unix.Flock(int(f.Fd()), unix.LOCK_UN)
		_ = f.Close()
	}, nil
}
