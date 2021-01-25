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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jamesl33/cbtools-autobench/value"
)

// TODO (jamesl33) We should print/display the cluster/backup client version.

// Report is the benchmark report which will be printed to stdout upon completion of the benchmarks.
type Report struct {
	Cluster      *value.ClusterBlueprint      `json:"cluster,omitempty"`
	BackupClient *value.BackupClientBlueprint `json:"backup_client,omitempty"`
	CBM          *value.CBMConfig             `json:"cbbackupmgr,omitempty"`
	Stats        *value.Stats                 `json:"bucket_stats,omitempty"`
	Overview     *Overview                    `json:"overview,omitempty"`
	Rundown      Rundown                      `json:"rundown,omitempty"`
	Logs         *Logs                        `json:"logs,omitempty"`
}

// NewReport creates a new report with the provided options.
func NewReport(options Options) *Report {
	return &Report{
		Cluster:      options.Blueprint.Cluster,
		Stats:        options.Stats,
		BackupClient: options.Blueprint.BackupClient,
		CBM:          options.CBMConfig,
		Overview:     NewOverview(options),
		Rundown:      NewRundown(options),
		Logs:         NewLogs(options),
	}
}

// String returns a string representation of the report. Components which are empty/unused will be omitted in a similar
// fashion to that of the 'omitempty' tag.
func (r *Report) String() string {
	buffer := &bytes.Buffer{}

	if r.Cluster != nil {
		fmt.Fprintf(buffer, "%s\n\n", r.Cluster)
	}

	if r.Stats != nil {
		fmt.Fprintf(buffer, "%s\n\n", r.Stats)
	}

	if r.BackupClient != nil {
		fmt.Fprintf(buffer, "%s\n\n", r.BackupClient)
	}

	if r.CBM != nil {
		fmt.Fprintf(buffer, "%s\n\n", r.CBM)
	}

	if r.Overview != nil {
		fmt.Fprintf(buffer, "%s\n\n", r.Overview)
	}

	if r.Rundown != nil {
		fmt.Fprintf(buffer, "%s\n\n", r.Rundown)
	}

	if r.Logs != nil {
		fmt.Fprintf(buffer, "%s\n", r.Logs)
	}

	return strings.TrimSpace(buffer.String())
}

// Print displays a string representation of the report, this is either a human readable form or standard JSON.
func (r *Report) Print(jsonOut bool) error {
	if !jsonOut {
		fmt.Printf("%s\n", r)
		return nil
	}

	rJSON, err := json.Marshal(r)
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", rJSON)

	return nil
}
