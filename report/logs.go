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

package report

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

// Logs is the component which displays information relating to the logs that were collected after completing the
// benchmarking process.
type Logs struct {
	Cluster []string
	Backup  string
}

// NewLogs creates a new 'Logs' component with the provided options.
func NewLogs(options Options) *Logs {
	if len(options.ClusterLogs) == 0 && options.BackupLogs == "" {
		return nil
	}

	return &Logs{
		Cluster: options.ClusterLogs,
		Backup:  options.BackupLogs,
	}
}

// String returns a string representation of the 'Logs' component which will be output in the report.
func (l *Logs) String() string {
	var (
		buffer = &bytes.Buffer{}
		writer = tabwriter.NewWriter(buffer, 4, 0, 1, ' ', tabwriter.Debug)
	)

	fmt.Fprintln(buffer, "| Logs\n| ----")
	fmt.Fprintf(writer, "| Path\t\n")

	for _, path := range l.Cluster {
		fmt.Fprintf(writer, "| %s\t\n", filepath.Base(path))
	}

	fmt.Fprintf(writer, "| %s\t\n", filepath.Base(l.Backup))

	_ = writer.Flush()

	return strings.TrimSpace(buffer.String())
}
