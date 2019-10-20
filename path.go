package trout

import (
	"strings"
)

func processPath(slices []string) (parts []string, names []string, indices []int, typs []int) {
	if len(slices) == 0 {
		panic("path is empty")
	}

	for i, s := range slices {
		index := -1
		typ := -1
		colon := strings.LastIndex(s, ":")
		wildcard := strings.LastIndex(s, "*")
		if colon > wildcard {
			index = colon
			typ = 0
		} else if colon < wildcard && i == len(slices)-1 {
			index = wildcard
			typ = 1
		}

		if index != -1 && index != len(s)-1 {
			parts = append(parts, s[:index])
			names = append(names, s[index+1:])
		} else {
			parts = append(parts, s)
			names = append(names, "")
		}
		indices = append(indices, index)
		typs = append(typs, typ)
	}

	return
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")

	if path == "" {
		return []string{""}
	}

	slices := strings.Split(path, "/")
	var nSlices []string

	for _, s := range slices {
		if len(s) > 0 {
			nSlices = append(nSlices, s)
		}
	}

	return nSlices
}
