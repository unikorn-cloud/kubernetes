/*
Copyright 2022-2024 EscherCloud.

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

package main

import (
	"os"

	unikornscheme "github.com/spjmurray/unikorn/generated/clientset/unikorn/scheme"
	"github.com/spjmurray/unikorn/pkg/cmd"

	"k8s.io/client-go/kubernetes/scheme"
)

func main() {
	// Imbue the Kubernetes client library with knowledge of our types.
	if err := unikornscheme.AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}

	c := cmd.Generate()

	if err := c.Execute(); err != nil {
		os.Exit(1)
	}
}
