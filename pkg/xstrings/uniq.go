package xstrings

type Comparable interface{ ~int | ~int64 | ~string }

func UniqueSlice[T Comparable](s []T) []T {
	keys := make(map[T]bool)
	list := []T{}
	for _, entry := range s {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
