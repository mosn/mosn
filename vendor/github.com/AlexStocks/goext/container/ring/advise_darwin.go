package gxring

// DontNeed is a NOP that returns nil. On linux it advises
// the system about memory no longer needed.
func (b *Buffer) DontNeed(c Cursor) error {
	return nil
}
