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

package util

import (
	"strings"
	"text/template"

	"github.com/spjmurray/unikorn/pkg/constants"

	"k8s.io/kubectl/pkg/util/templates"
)

// DynamicTemplateOptions allows some parameters to be passed into help text
// and that text to be templated so it will update automatically when the
// options do.
type DynamicTemplateOptions struct {
	// Application is the application name as defined by argv[0].
	Application string
}

// newDynamicTemplateOptions returns am intialiized template options struct.
func newDynamicTemplateOptions() *DynamicTemplateOptions {
	return &DynamicTemplateOptions{
		Application: constants.Application,
	}
}

// templatedString allows dynamic templating e.g. variable expansion, of
// strings, typically in help text examples.
func templatedString(s string, data any) string {
	t := template.New("root")

	t, err := t.Parse(s)
	if err != nil {
		panic(err)
	}

	out := &strings.Builder{}

	if err := t.Execute(out, data); err != nil {
		panic(err)
	}

	return out.String()
}

// templatedExample applies a templating function to the example string so
// we can make the text dyanamic.  It also applies standard Kubernetes
// formatting rules after the templating step.
func TemplatedExample(s string) string {
	s = templatedString(s, newDynamicTemplateOptions())

	return templates.Examples(s)
}
