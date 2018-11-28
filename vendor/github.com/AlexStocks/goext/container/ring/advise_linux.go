package gxring

import "syscall"

// DontNeed advises the system about memory no longer needed.
func (b *Buffer) DontNeed(c Cursor) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if c.offset < b.first {
		// c not caught up.
		return nil
	}
	var (
		start    = b.first % b.capacity
		end      = c.offset % b.capacity
		wrapping = end <= start
	)

	if s := start % pageSize; s != 0 {
		start = start - s + pageSize
		// It is no longer guaranteed that start < b.capacity.
		// Also, even when !wrapping it is now possible that
		// start > end.
	}

	if !wrapping {
		return b.dontNeed(start, end)
	}

	if start < b.capacity {
		if err := b.dontNeed(start, b.capacity); err != nil {
			return err
		}
	}
	return b.dontNeed(0, end)
}

func (b *Buffer) dontNeed(i, j uint64) error {
	// Since i is rounded up to the nearest multiple of
	// pageSize before the call, i ≥ j is not an error.
	// i == j can happen when i = j = 0.
	if i >= j {
		return nil
	}
	return syscall.Madvise(b.data[i:j], syscall.MADV_DONTNEED)
}
