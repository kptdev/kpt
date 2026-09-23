// Copyright 2026 The kpt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kptfile

// SetLabels sets the labels in the Kptfile
func (kf *KptfileKubeObject) SetLabels(labels map[string]string) {
	for k, v := range labels {
		_ = kf.SetLabel(k, v)
	}

	existing := kf.GetLabels()
	for k := range existing {
		if _, ok := labels[k]; !ok {
			_ = kf.RemoveLabel(k)
		}
	}
}

// SetAnnotations sets the annotations in the Kptfile
func (kf *KptfileKubeObject) SetAnnotations(annotations map[string]string) {
	for k, v := range annotations {
		_ = kf.SetAnnotation(k, v)
	}

	existing := kf.GetAnnotations()
	for k := range existing {
		if _, ok := annotations[k]; !ok {
			_ = kf.RemoveAnnotation(k)
		}
	}
}
