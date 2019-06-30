package proteus

// Wrapper is now a no-op func that exists for backward compatibility. It is now deprecated and will be removed in the
// 1.0 release of proteus.
func Wrap(sqle Wrapper) Wrapper {
	return sqle
}
