package cmp

func Errors(e1, e2 error) bool {
	if e1 == nil || e2 == nil {
		if e1 != nil || e2 != nil {
			return false
		}
		return true
	}
	return e1.Error() == e2.Error()
}
