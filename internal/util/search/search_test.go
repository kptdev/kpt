package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type patternTest struct {
	name         string
	fieldValue   string
	valueRegex   string
	patternRegex string
	expected     string
	errMsg       string
}

var resolvePatternCases = []patternTest{
	{
		name:         "resolve pattern 1",
		fieldValue:   "foo1-dev-bar1-us-east-1-baz1",
		valueRegex:   `(\w+)-dev-(\w+)-us-east-1-(\w+)`,
		patternRegex: "${1}-prod-${2}-us-central-1-${3}",
		expected:     "foo1-prod-bar1-us-central-1-baz1",
	},
	{
		name:         "resolve pattern 2",
		fieldValue:   "foo1-dev-bar1-us-east-1-baz1",
		valueRegex:   `(\w+)-dev-(\w+)-us-east-1-(\w+)`,
		patternRegex: "kpt-set: ${1}-${environment}-${2}-${region1}-${3}",
		expected:     "kpt-set: foo1-${environment}-bar1-${region1}-baz1",
	},
	{
		name:         "resolve pattern 3",
		fieldValue:   "foo1-dev-bar1-us-east-1-baz1",
		valueRegex:   `(\w+)-dev-(\w+)-us-east-1-(\w+)`,
		patternRegex: "some-irrelevant-string",
		expected:     "some-irrelevant-string",
	},
	{
		name:         "resolve pattern 4",
		fieldValue:   "foo1-dev-bar1-us-east-1-baz1",
		valueRegex:   "some-irrelevant-string-${1}",
		patternRegex: "kpt-set: ${1}-${environment}-${2}-${region1}-${3}",
		errMsg:       "unable to resolve capture groups",
	},
	{
		name:         "resolve pattern 5",
		fieldValue:   "foo1-dev-bar1-us-east-1-baz1",
		valueRegex:   `(\w+)-dev-(\w+)-us-east-1-(\w+)`,
		patternRegex: "kpt-set: ${1}-${environment}-${2}-${region1}-${3}-extra-${4}",
		errMsg:       "unable to resolve capture groups",
	},
	{
		name:         "resolve pattern 6",
		fieldValue:   "foo1-dev-bar1-us-east-1-baz1",
		valueRegex:   "abc-(*",
		patternRegex: "kpt-set: ${1}-${environment}-${2}-${region1}-${3}-extra-${4}",
		errMsg:       "failed to compile input pattern",
	},
}

func TestResolvePattern(t *testing.T) {
	for _, tests := range [][]patternTest{resolvePatternCases} {
		for i := range tests {
			test := tests[i]
			t.Run(test.name, func(t *testing.T) {
				res, err := resolvePattern(test.fieldValue, test.valueRegex, test.patternRegex)
				if test.errMsg != "" {
					if !assert.NotNil(t, err) {
						t.FailNow()
					}
					if !assert.Contains(t, err.Error(), test.errMsg) {
						t.FailNow()
					}
				}
				if !assert.Equal(t, test.expected, res) {
					t.FailNow()
				}
			})
		}
	}
}
