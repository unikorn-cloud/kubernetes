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
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	spdx_common "github.com/spdx/tools-golang/spdx/v2/common"
	spdx "github.com/spdx/tools-golang/spdx/v2/v2_3"

	unikornv1 "github.com/spjmurray/unikorn/pkg/apis/unikorn/v1alpha1"

	coreunikornv1 "github.com/spjmurray/unikorn-core/pkg/apis/unikorn/v1alpha1"

	"sigs.k8s.io/yaml"
)

// parseResourceFile loads the YAML manifest from the path, and unmarshals it into
// a list of the provided template type.
func parseResourceFile[T any](path string) ([]T, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(string(data), "\n---\n")

	result := make([]T, len(parts))

	for i, part := range parts {
		if err := yaml.Unmarshal([]byte(part), &result[i]); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// parseResources loads all the manifest files, and returns list types of application bundles
// and helm applications.
func parseResources() (*unikornv1.ControlPlaneApplicationBundleList, *unikornv1.KubernetesClusterApplicationBundleList, *coreunikornv1.HelmApplicationList, error) {
	controlPlaneApplicationBundles, err := parseResourceFile[unikornv1.ControlPlaneApplicationBundle]("charts/unikorn/templates/controlplaneapplicationbundles.yaml")
	if err != nil {
		return nil, nil, nil, err
	}

	kubernetesClusterApplicationBundles, err := parseResourceFile[unikornv1.KubernetesClusterApplicationBundle]("charts/unikorn/templates/kubernetesclusterapplicationbundles.yaml")
	if err != nil {
		return nil, nil, nil, err
	}

	applications, err := parseResourceFile[coreunikornv1.HelmApplication]("charts/unikorn/templates/applications.yaml")
	if err != nil {
		return nil, nil, nil, err
	}

	return &unikornv1.ControlPlaneApplicationBundleList{Items: controlPlaneApplicationBundles}, &unikornv1.KubernetesClusterApplicationBundleList{Items: kubernetesClusterApplicationBundles}, &coreunikornv1.HelmApplicationList{Items: applications}, nil
}

// getApplication looks up an application by name.
func getApplication(name string, applications *coreunikornv1.HelmApplicationList) (*coreunikornv1.HelmApplication, error) {
	for i, application := range applications.Items {
		if application.Name == name {
			return &applications.Items[i], nil
		}
	}

	//nolint:goerr113
	return nil, fmt.Errorf("FATAL: unable to locate application %s", name)
}

// See: https://helm.sh/docs/topics/charts/
type helmChartInfoMaintainer struct {
	Name  string  `json:"name"`
	Email *string `json:"email"`
	URL   *string `json:"url"`
}

// See: https://helm.sh/docs/topics/charts/
type helmChartInfoDependency struct {
	Name       string   `json:"name"`
	Version    string   `json:"version"`
	Repository *string  `json:"repository"`
	Condition  *string  `json:"condition"`
	Tags       []string `json:"tags"`
	//nolint: tagliatelle
	ImportValues []string `json:"import-values"`
	Alias        *string  `json:"alias"`
}

// See: https://helm.sh/docs/topics/charts/
type helmChartInfoStruct struct {
	APIVersion   string                    `json:"apiVersion"`
	Name         string                    `json:"name"`
	Version      string                    `json:"version"`
	KubeVersion  *string                   `json:"kubeVersion"`
	Description  *string                   `json:"description"`
	Type         *string                   `json:"type"`
	Keywords     []string                  `json:"keywords"`
	Home         *string                   `json:"home"`
	Sources      []string                  `json:"sources"`
	Dependencies []helmChartInfoDependency `json:"dependencies"`
	Maintainers  []helmChartInfoMaintainer `json:"maintainers"`
	Icon         *string                   `json:"icon"`
	AppVersion   *string                   `json:"appVersion"`
	Deprecated   *bool                     `json:"deprecated"`
	Annotations  map[string]string         `json:"annotations"`
}

// helmChartInfo grabs information about the given application's helm chart.
func helmChartInfo(repo, chart, version string) (*helmChartInfoStruct, error) {
	command := exec.Command("helm", "show", "chart", chart,
		"--repo", repo,
		"--version", version)

	data, err := command.Output()
	if err != nil {
		return nil, err
	}

	info := &helmChartInfoStruct{}

	if err := yaml.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return info, nil
}

// pruneEmptyStrings helper to remove empty strings from a string slice.
func pruneEmptyStrings(s []string) []string {
	var r []string

	for _, v := range s {
		if v != "" {
			r = append(r, v)
		}
	}

	return r
}

// generatePackage does some helm wizardry to lookup application package details.
//
//nolint:cyclop
func generatePackage(repo, chart, version string) ([]*spdx.Package, error) {
	repoURL, err := url.Parse(repo)
	if err != nil {
		return nil, err
	}

	// Ignore things that don't look like real repos, e.g. file://
	if repoURL.Scheme != "https" {
		return nil, nil
	}

	// Sloooooooow..... zzzz
	info, err := helmChartInfo(repo, chart, version)
	if err != nil {
		return nil, err
	}

	// See: https://spdx.dev/spdx-specification-20-web-version/ section 3.2.4.
	re := regexp.MustCompile(`[^a-z0-9.-]`)

	id := []string{
		"HelmChart",
		re.ReplaceAllString(repoURL.Host, "-"),
		re.ReplaceAllString(strings.TrimPrefix(repoURL.Path, "/"), "-"),
		chart,
		version,
	}

	idString := strings.Join(pruneEmptyStrings(id), "-")

	// Fill in all the basics we know are mandatory.
	// NOTE: some charts will use a wildcaed version, so it's only as
	// up to date as when the SBOM was created.
	p := &spdx.Package{
		PackageSPDXIdentifier:   spdx_common.ElementID(idString),
		PackageName:             chart,
		PackageVersion:          info.Version,
		PackageDownloadLocation: repo,
		PackageLicenseDeclared:  "NONE",
		PrimaryPackagePurpose:   "INSTALL",
	}

	// Add in any optional fields.
	if info.Home != nil {
		p.PackageHomePage = *info.Home
	}

	if info.AppVersion != nil {
		p.PackageComment = "Application Version: " + *info.AppVersion
	}

	if info.Annotations != nil {
		if license, ok := info.Annotations["artifacthub.io/license"]; ok {
			p.PackageLicenseDeclared = license
		}
	}

	result := []*spdx.Package{
		p,
	}

	// Recursively walk down the dependency tree...
	for _, dependency := range info.Dependencies {
		if dependency.Repository == nil {
			continue
		}

		d, err := generatePackage(*dependency.Repository, dependency.Name, dependency.Version)
		if err != nil {
			return nil, err
		}

		result = append(result, d...)
	}

	return result, nil
}

// generateSBOM does the actual meat!
func generateSBOM(name string, spec *unikornv1.ApplicationBundleSpec, applications *coreunikornv1.HelmApplicationList) error {
	document := &spdx.Document{
		SPDXVersion:       spdx.Version,
		DataLicense:       spdx.DataLicense,
		SPDXIdentifier:    "DOCUMENT",
		DocumentName:      name,
		DocumentNamespace: "unikorn.spjmurray.co.uk",
		CreationInfo: &spdx.CreationInfo{
			Creators: []spdx_common.Creator{
				{
					Creator:     "EscherCloud AI",
					CreatorType: "Organization",
				},
				{
					Creator:     "unikorn",
					CreatorType: "Tool",
				},
			},
			Created: time.Now().UTC().Format(time.RFC3339),
		},
	}

	for _, applicationRef := range spec.Applications {
		application, err := getApplication(*applicationRef.Reference.Name, applications)
		if err != nil {
			return err
		}

		for _, version := range application.Spec.Versions {
			// Skip things that are pulled directly from GitHub as I cannot be bothered
			// to figure out how to get the Chart.yaml file... something, something
			// rawcontent.
			if version.Path != nil {
				continue
			}

			p, err := generatePackage(*version.Repo, *version.Chart, *version.Version)
			if err != nil {
				return err
			}

			// TODO: we should probably deduplicate this, just in case.
			document.Packages = append(document.Packages, p...)
		}
	}

	data, err := json.Marshal(document)
	if err != nil {
		return err
	}

	file, err := os.Create("sboms/" + name + ".spdx")
	if err != nil {
		return err
	}

	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return err
	}

	return nil
}

// main gets the necessary helm template definitions then generates an SBOM for each
// application bundle.
func main() {
	controlPlaneApplicationBundles, kubernetesClusterApplicationBundles, applications, err := parseResources()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := os.MkdirAll("sboms", 0775); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for i := range controlPlaneApplicationBundles.Items {
		bundle := &controlPlaneApplicationBundles.Items[i]

		if err := generateSBOM(bundle.Name, &bundle.Spec, applications); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	for i := range kubernetesClusterApplicationBundles.Items {
		bundle := &kubernetesClusterApplicationBundles.Items[i]

		if err := generateSBOM(bundle.Name, &bundle.Spec, applications); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}
