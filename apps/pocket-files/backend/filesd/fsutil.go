package main

import "syscall"

// statfsNumbers returns total and free bytes for a directory. Real providers use
// it for honest capacity; mock volumes report their declared nominal capacity.
func statfsNumbers(path string) (total, avail, free int64, err error) {
	var st syscall.Statfs_t
	if err = syscall.Statfs(path, &st); err != nil {
		return 0, 0, 0, err
	}
	bsize := int64(st.Bsize)
	total = bsize * int64(st.Blocks)
	free = bsize * int64(st.Bfree)
	avail = bsize * int64(st.Bavail)
	return total, avail, free, nil
}
