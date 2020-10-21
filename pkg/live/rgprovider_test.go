// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
)

func TestResourceGroupProvider_ManifestReader(t *testing.T) {
	testCases := map[string]struct {
		args     []string
		reader   io.Reader
		expected string
		isError  bool
	}{
		"No args or reader is an error": {
			args:     []string{},
			reader:   nil,
			expected: "",
			isError:  true,
		},
		"More than one args is an error": {
			args:     []string{"dir-1", "dir-2"},
			reader:   nil,
			expected: "",
			isError:  true,
		},
		"No args returns stream reader": {
			args:     []string{},
			reader:   strings.NewReader("foo"),
			expected: "*ResourceGroupStreamManifestReader",
			isError:  false,
		},
		"One arg returns path reader": {
			args:     []string{"/fake-directory-str"},
			reader:   nil,
			expected: "*ResourceGroupPathManifestReader",
			isError:  false,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			tf := cmdtesting.NewTestFactory().WithNamespace("test-ns")
			defer tf.Cleanup()
			rgProvider := NewResourceGroupProvider(tf)
			actual, err := rgProvider.ManifestReader(tc.reader, tc.args)
			// Check if there should be an error
			if tc.isError {
				if err == nil {
					t.Fatalf("expected error but received none")
				}
				return
			}
			assert.NoError(t, err)
			if tc.expected != getType(actual) {
				t.Errorf("expected ManifestReader type (%s), got (%s)", tc.expected, getType(actual))
			}
		})
	}
}

func getType(myvar interface{}) string {
	t := reflect.TypeOf(myvar)
	if t.Kind() == reflect.Ptr {
		return "*" + t.Elem().Name()
	}
	return t.Name()
}
