package cmp

type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 |
		~string
}

func Compare[T Ordered](x, y T) int {
	switch {
	case x < y:
		return -1
	case x > y:
		return +1
	default:
		return 0
	}
}

func Or[T comparable](vals ...T) T {
	var zero T
	for _, val := range vals {
		if val != zero {
			return val
		}
	}
	return zero
}
