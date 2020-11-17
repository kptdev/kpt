package search

import (
	"testing"

	"github.com/golangplus/testing/assert"
)

type test struct {
	name          string
	byPath        string
	traversedPath string
	shouldMatch   bool
}

var tests = []test{
	{
		name:          "simple path match",
		byPath:        "a.b.c",
		traversedPath: "a.b.c",
		shouldMatch:   true,
	},
	{
		name:          "simple path no match",
		byPath:        "a.b.c",
		traversedPath: "a.c.b",
		shouldMatch:   false,
	},
	{
		name:          "simple path match with *",
		byPath:        "a.*.c.*",
		traversedPath: "a.b.c.d",
		shouldMatch:   true,
	},
	{
		name:          "simple path no match with *",
		byPath:        "a.*.c.*",
		traversedPath: "a.b.c",
		shouldMatch:   false,
	},
	{
		name:          "simple array path match",
		byPath:        "a.c[0]",
		traversedPath: "a.c[0]",
		shouldMatch:   true,
	},
}

func TestPathMatch(t *testing.T) {
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			sr := SearchReplace{
				ByPath: test.byPath,
			}
			actual := sr.pathMatch(test.traversedPath)
			if !assert.Equal(t, "", actual, test.shouldMatch) {
				t.FailNow()
			}
		})
	}
}
