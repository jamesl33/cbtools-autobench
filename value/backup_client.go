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
	"strings"
	"text/tabwriter"
)

// BackupClientBlueprint encapsulates the available configuration for the backup client which will be provisioned by the
// 'provison' sub-command.
type BackupClientBlueprint struct {
	// Host is the hostname/address of the node
	Host string `yaml:"host,omitempty"`

	// PackagePath is the path to a local package. This package will be secure copied to the backup client and installed
	// instead of downloading the build from latest builds.
	//
	// NOTE: No validation takes place to ensure the package is valid for the current distribution; that's on you...
	PackagePath string `yaml:"package_path,omitempty"`
}

// MarshalJSON returns a JSON representation of the backup blueprint which will be displayed in the report.
func (b *BackupClientBlueprint) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Host    string `json:"host,omitempty"`
		Version string `json:"version,omitempty"`
	}{
		Host:    b.Host,
		Version: extractBuild(b.PackagePath),
	})
}

// String returns a human readable string representation of the backup blueprint which will be displayed in the report.
func (b *BackupClientBlueprint) String() string {
	var (
		buffer = &bytes.Buffer{}
		writer = tabwriter.NewWriter(buffer, 4, 0, 1, ' ', tabwriter.Debug)
	)

	fmt.Fprintln(buffer, "| Backup Client\n| -------------")
	fmt.Fprintf(writer, "| Version\t Host\t\n")
	fmt.Fprintf(writer, "| %s\t %s\t\n", extractBuild(b.PackagePath), b.Host)

	_ = writer.Flush()

	return strings.TrimSpace(buffer.String())
}
