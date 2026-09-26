package json

import (
	"reflect"
	"sync"
)

type TextAppender interface {
	AppendText([]byte) ([]byte, error)
}

type textAppender = TextAppender

func typeFor[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

func onceValue[T any](f func() T) func() T {
	var (
		once sync.Once
		v    T
	)
	return func() T {
		once.Do(func() {
			v = f()
		})
		return v
	}
}
