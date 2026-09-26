package json

import "reflect"

func typeFor[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}
