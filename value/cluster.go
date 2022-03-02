// Copyright 2021 Couchbase Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package value

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"text/tabwriter"
)

// ClusterBlueprint encapsulates the configuration for the Couchbase Cluster which will be provisioned by the
// 'provision' sub-command.
type ClusterBlueprint struct {
	// PackagePath is the path to a local package. This package will be secure copied to each cluster node and installed
	// instead of downloading the build from latest builds.
	//
	// NOTE: No validation takes place to ensure the package is valid for the current distribution; that's on you...
	PackagePath string `yaml:"package_path,omitempty"`

	// Nodes is the list of node blueprints which will be used to create the cluster.
	Nodes []*NodeBlueprint `yaml:"nodes,omitempty"`

	// Bucket is the blueprint for the bucket that will be created once the cluster is provisioned.
	Bucket *BucketBlueprint `yaml:"bucket,omitempty"`

	// DeveloperPreview is a boolean which indicates whether or not developer preview should be enabled on the
	// cluster.
	DeveloperPreview bool `yaml:"developer_preview,omitempty"`
}

// MarshalJSON returns a JSON representation of the cluster blueprint which will be displayed in the report.
func (c *ClusterBlueprint) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Version          string           `json:"version,omitempty"`
		Nodes            []*NodeBlueprint `json:"nodes,omitempty"`
		Bucket           *BucketBlueprint `json:"bucket,omitempty"`
		DeveloperPreview bool             `json:"developer_preview,omitempty"`
	}{
		Version:          extractBuild(c.PackagePath),
		Nodes:            c.Nodes,
		Bucket:           c.Bucket,
		DeveloperPreview: c.DeveloperPreview,
	})
}

// String returns a human readable string representation of the cluster blueprint which will be displayed in the report.
func (c *ClusterBlueprint) String() string {
	var (
		buffer = &bytes.Buffer{}
		writer = tabwriter.NewWriter(buffer, 4, 0, 1, ' ', tabwriter.Debug)
	)

	fmt.Fprintln(buffer, "| Cluster\n| -------")
	fmt.Fprintf(writer, "| Node\t Version\t Host\t Developer Preview\t\n")

	for index, node := range c.Nodes {
		fmt.Fprintf(writer, "| %d\t %s\t %s\t %t\t\n", index+1, extractBuild(c.PackagePath), node.Host,
			c.DeveloperPreview)
	}

	_ = writer.Flush()

	fmt.Fprintf(buffer, "\n%s", c.Bucket)

	return strings.TrimSpace(buffer.String())
}

// extractBuild will extract the build number from the provided string. Returns 'unknown' in the event that we're unable
// to determine the version.
func extractBuild(s string) string {
	version := "unknown"
	if match := regexp.MustCompile(RegexBuildID).FindStringSubmatch(s); match != nil {
		version = match[0]
	}

	return version
}
