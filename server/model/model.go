// Package model define the types stored in the database.
package model

import (
	"encoding/binary"
	"fmt"
	"io"
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

func equalInstructorClassSlice(a, b []InstructorClass) bool {
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

func reverse(fn func(i, j int) bool) func(i, j int) bool {
	return func(i, j int) bool {
		return fn(j, i)
	}
}

func noReverse(fn func(i, j int) bool) func(i, j int) bool {
	return fn
}

func SortKeyReverse(key string) (string, func(func(int, int) bool) func(int, int) bool) {
	switch {
	case key == "":
		return "", noReverse
	case key[0] == '-':
		return key[1:], reverse
	default:
		return key, noReverse
	}
}
