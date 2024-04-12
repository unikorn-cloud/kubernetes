/*
Copyright 2024 the Unikorn Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package providers

import (
	"maps"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	MetdataDomain = "provider.unikorn-cloud.org"
)

func GetAnnotations(resource metav1.Object) map[string]string {
	annotations := maps.Clone(resource.GetAnnotations())

	maps.DeleteFunc(annotations, func(k, v string) bool {
		return !strings.Contains(k, MetdataDomain)
	})

	return annotations
}
