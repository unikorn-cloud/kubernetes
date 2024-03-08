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

func (l FlavorList) Len() int {
	return len(l)
}

func (l FlavorList) Less(i, j int) bool {
	// Sort by GPUs, we want these to have precedence, we are selling GPUs
	// after all.  Those with the smallest number of GPUs go first, we want to
	// prevent over provisioning.
	if l[i].GPUs < l[j].GPUs {
		return true
	}

	// If the GPUs are the same, sort by CPUs.
	if l[i].CPUs < l[j].CPUs {
		return true
	}

	return false
}

func (l FlavorList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l ImageList) Len() int {
	return len(l)
}

func (l ImageList) Less(i, j int) bool {
	return l[i].Created.Before(l[j].Created)
}

func (l ImageList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}
