package fnruntime

import (
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/types"
	kptfile "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestIsMatch(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		selector kptfile.Selector
		expected bool
	}{
		{
			name: "kind match",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
spec:
  replicas: 3`,
			selector: kptfile.Selector{
				Kind: "Deployment",
			},
			expected: true,
		},
		{
			name: "name match",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
spec:
  replicas: 3`,
			selector: kptfile.Selector{
				Name: "nginx-deployment",
			},
			expected: true,
		},
		{
			name: "namespace match",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: staging
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
spec:
  replicas: 3`,
			selector: kptfile.Selector{
				Namespace: "staging",
			},
			expected: true,
		},
		{
			name: "apiVersion match",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
spec:
  replicas: 3`,
			selector: kptfile.Selector{
				APIVersion: "apps/v1",
			},
			expected: true,
		},
		{
			name: "GVKNN match",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: staging
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
spec:
  replicas: 3`,
			selector: kptfile.Selector{
				Name:       "nginx-deployment",
				Namespace:  "staging",
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			expected: true,
		},
		{
			name: "namespace not matched but rest did",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: staging
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
spec:
  replicas: 3`,
			selector: kptfile.Selector{
				Name:       "nginx-deployment",
				Namespace:  "prod",
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			expected: false,
		},
		{
			name: "packagePath match",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
    internal.config.kubernetes.io/package-path: "/path/to/root/pkg/db"
spec:
  replicas: 3`,
			selector: kptfile.Selector{
				PackagePath: "./db",
			},
			expected: true,
		},
	}

	for i := range tests {
		tc := tests[i]
		t.Run(tc.name, func(t *testing.T) {
			node, err := yaml.Parse(tc.input)
			assert.NoError(t, err)
			var rootPackagePath string
			if tc.selector.PackagePath != "" {
				rootPackagePath = "/path/to/root/pkg"
			}
			actual, err := isMatch(node, tc.selector, &SelectionContext{RootPackagePath: types.UniquePath(rootPackagePath)})
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
