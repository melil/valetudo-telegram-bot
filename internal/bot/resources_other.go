//go:build !linux

package bot

import "errors"

func getDiskSpaceInfo(path string) (total uint64, free uint64, used uint64, err error) {
	return 0, 0, 0, errors.New("disk space info not supported on this platform")
}
