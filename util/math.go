package util

func Abs(val int) int {
	if val > 0 {
		return val
	}
	return -val
}

func Max(first int, second int) int {
	if first > second {
		return first
	}
	return second
}

func Min(first int, second int) int {
	if first < second {
		return first
	}
	return second
}
