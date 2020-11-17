package search

import (
	"strings"
)

// pathMatch checks if the traversed yaml path matches with the user input path
// checks if user input path is valid
func (sr *SearchReplace) pathMatch(yamlPath string) bool {
	if sr.ByPath == "" {
		return false
	}
	inputElems := strings.Split(sr.ByPath, PathDelimiter)
	traversedElems := strings.Split(strings.Trim(yamlPath, PathDelimiter), PathDelimiter)
	if len(inputElems) != len(traversedElems) {
		return false
	}
	for i, inputElem := range inputElems {
		if inputElem != "*" && inputElem != traversedElems[i] {
			return false
		}
	}
	return true
}

// isAbsPath checks if input path is absolute and not a path expression
// only supported path format is e.g. foo.bar.baz
func isAbsPath(path string) bool {
	pathElem := strings.Split(path, PathDelimiter)
	if len(pathElem) == 0 {
		return false
	}
	for _, elem := range pathElem {
		// more checks can be added in future
		if elem == "" || strings.Contains(elem, "*") {
			return false
		}
	}
	return true
}
