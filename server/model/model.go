// Package model define the types stored in the database.
package model

import (
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
)

func equalIntSlice(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func hashValue(w io.Writer, v interface{}) {
	switch v := v.(type) {
	case string:
		binary.Write(w, binary.LittleEndian, len(v))
		io.WriteString(w, v)
	case []int:
		binary.Write(w, binary.LittleEndian, len(v))
		for _, i := range v {
			binary.Write(w, binary.LittleEndian, int64(i))
		}
	case []string:
		binary.Write(w, binary.LittleEndian, len(v))
		for _, s := range v {
			binary.Write(w, binary.LittleEndian, len(s))
			io.WriteString(w, s)
		}
	case int:
		binary.Write(w, binary.LittleEndian, int64(v))
	default:
		err := binary.Write(w, binary.LittleEndian, v)
		if err != nil {
			panic(fmt.Sprintf("cannot hash value of type %T", v))
		}
	}
}

func reverseSlice(s interface{}) {
	size := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, size-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
}

func noReverseSlice(s interface{}) {}

func SortKeyReverse(key string) (string, func(s interface{})) {
	switch {
	case key == "":
		return "", noReverseSlice
	case key[0] == '-':
		return key[1:], reverseSlice
	default:
		return key, noReverseSlice
	}
}
