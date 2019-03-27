package shm

func Alloc(name string, size int) (*ShmSpan, error) {
	path := path(name)

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)

	if err != nil {
		return nil, err
	}

	defer f.Close()

	if err := f.Truncate(int64(size)); err != nil {
		return nil, err
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, size, syscall.PROT_WRITE, syscall.MAP_SHARED)

	if err != nil {
		return nil, err
	}

	return NewShmSpan(path, data), nil
}

func DeAlloc(span *ShmSpan) error {
	os.Remove(span.path)
	return syscall.Munmap(span.origin)
}
