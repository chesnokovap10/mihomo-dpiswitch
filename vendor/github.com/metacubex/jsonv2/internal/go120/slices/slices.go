package slices

import (
	"sort"

	"github.com/metacubex/jsonv2/internal/go120/cmp"
	"github.com/metacubex/jsonv2/internal/go120/iter"
)

func Equal[S ~[]E, E comparable](s1 S, s2 S) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := range s1 {
		if s1[i] != s2[i] {
			return false
		}
	}
	return true
}

func Contains[S ~[]E, E comparable](s S, v E) bool {
	for _, e := range s {
		if e == v {
			return true
		}
	}
	return false
}

func Compare[S ~[]E, E cmp.Ordered](s1 S, s2 S) int {
	for i := 0; i < len(s1) && i < len(s2); i++ {
		if c := cmp.Compare(s1[i], s2[i]); c != 0 {
			return c
		}
	}
	return cmp.Compare(len(s1), len(s2))
}

func CompareFunc[S1 ~[]E1, S2 ~[]E2, E1, E2 any](s1 S1, s2 S2, compare func(E1, E2) int) int {
	for i := 0; i < len(s1) && i < len(s2); i++ {
		if c := compare(s1[i], s2[i]); c != 0 {
			return c
		}
	}
	return cmp.Compare(len(s1), len(s2))
}

func Sort[S ~[]E, E cmp.Ordered](s S) {
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
}

func SortFunc[S ~[]E, E any](s S, compare func(E, E) int) {
	sort.Slice(s, func(i, j int) bool { return compare(s[i], s[j]) < 0 })
}

func SortStableFunc[S ~[]E, E any](s S, compare func(E, E) int) {
	sort.SliceStable(s, func(i, j int) bool { return compare(s[i], s[j]) < 0 })
}

func BinarySearchFunc[S ~[]E, E, T any](x S, target T, compare func(E, T) int) (int, bool) {
	n := len(x)
	i := sort.Search(n, func(i int) bool { return compare(x[i], target) >= 0 })
	return i, i < n && compare(x[i], target) == 0
}

func Grow[S ~[]E, E any](s S, n int) S {
	if n < 0 {
		panic("cannot be negative")
	}
	if n -= cap(s) - len(s); n > 0 {
		s = append(s[:cap(s)], make([]E, n)...)[:len(s)]
	}
	return s
}

func Clone[S ~[]E, E any](s S) S {
	if s == nil {
		return nil
	}
	return append(S{}, s...)
}

func Concat[S ~[]E, E any](slices ...S) S {
	size := 0
	for _, s := range slices {
		size += len(s)
	}
	out := make(S, 0, size)
	for _, s := range slices {
		out = append(out, s...)
	}
	return out
}

func All[S ~[]E, E any](s S) iter.Seq2[int, E] {
	return func(yield func(int, E) bool) {
		for i, v := range s {
			if !yield(i, v) {
				return
			}
		}
	}
}

func Values[S ~[]E, E any](s S) iter.Seq[E] {
	return func(yield func(E) bool) {
		for _, v := range s {
			if !yield(v) {
				return
			}
		}
	}
}

func Backward[S ~[]E, E any](s S) iter.Seq2[int, E] {
	return func(yield func(int, E) bool) {
		for i := len(s) - 1; i >= 0; i-- {
			if !yield(i, s[i]) {
				return
			}
		}
	}
}

func Collect[E any](seq iter.Seq[E]) []E {
	return AppendSeq([]E(nil), seq)
}

func AppendSeq[S ~[]E, E any](s S, seq iter.Seq[E]) S {
	seq(func(v E) bool {
		s = append(s, v)
		return true
	})
	return s
}

func Sorted[E cmp.Ordered](seq iter.Seq[E]) []E {
	s := Collect(seq)
	Sort(s)
	return s
}

func SortedFunc[E any](seq iter.Seq[E], compare func(E, E) int) []E {
	s := Collect(seq)
	SortFunc(s, compare)
	return s
}

func SortedStableFunc[E any](seq iter.Seq[E], compare func(E, E) int) []E {
	s := Collect(seq)
	SortStableFunc(s, compare)
	return s
}
