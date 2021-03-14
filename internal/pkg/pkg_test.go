package pkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPkg(t *testing.T) {
	var tests = []struct {
		name        string
		inputPath   string
		uniquePath  string
		displayPath string
	}{
		{
			name:        "test1",
			inputPath:   ".",
			displayPath: ".",
		},
		{
			name:        "test2",
			inputPath:   "../",
			displayPath: "..",
		},
		{
			name:        "test3",
			inputPath:   "./foo/bar/",
			displayPath: "foo/bar",
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			p, err := New(test.inputPath)
			assert.NoError(t, err)
			assert.Equal(t, test.displayPath, string(p.DisplayPath))
		})
	}
}
