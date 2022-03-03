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
	"fmt"
	"strconv"
	"text/tabwriter"
)

// BucketBlueprint represents the configration for a bucket that will be created by the 'provision' sub-command.
type BucketBlueprint struct {
	VBuckets          uint16         `json:"vbuckets,omitempty" yaml:"vbuckets,omitempty"`
	Type              string         `json:"type,omitempty" yaml:"type,omitempty"`
	EvictionPolicy    string         `json:"eviction_policy,omitempty" yaml:"eviction_policy,omitempty"`
	Compact           bool           `json:"compact,omitempty" yaml:"compact,omitempty"`
	PiTREnabled       bool           `json:"pitr_enabled,omitempty" yaml:"pitr_enabled,omitempty"`
	PiTRGranularity   uint64         `json:"pitr_granularity,omitempty" yaml:"pitr_granularity,omitempty"`
	PiTRMaxHistoryAge uint64         `json:"pitr_max_history_age,omitempty" yaml:"pitr_max_history_age,omitempty"`
	Data              *DataBlueprint `json:"data,omitempty" yaml:"data,omitempty"`
}

// String returns a string representation of the blueprint which will be output in the report.
func (b *BucketBlueprint) String() string {
	var (
		buffer = &bytes.Buffer{}
		writer = tabwriter.NewWriter(buffer, 4, 0, 1, ' ', tabwriter.Debug)
	)

	vbuckets := "default"
	if b.VBuckets != 0 {
		vbuckets = strconv.Itoa(int(b.VBuckets))
	}

	bucketType := "default"
	if b.Type != "" {
		bucketType = b.Type
	}

	evictionPolicy := "default"
	if b.EvictionPolicy != "" {
		evictionPolicy = b.EvictionPolicy
	}

	pitrGranularity, pitrMaxHistoryAge := b.stringifyPiTRSettings()

	fmt.Fprintln(buffer, "| Bucket\n| ------")
	fmt.Fprintf(writer, "| vBuckets\t Type\t Eviction Policy\t PiTR Enabled\t PiTR Granularity\t PiTR Max History "+
		"Age\t Compact\t\n")
	fmt.Fprintf(writer, "| %s\t %s\t %s\t %t\t %s\t %s\t %t\t\n", vbuckets, bucketType, evictionPolicy, b.PiTREnabled,
		pitrGranularity, pitrMaxHistoryAge, b.Compact)

	_ = writer.Flush()

	fmt.Fprintf(buffer, "\n%s", b.Data)

	return buffer.String()
}

// stringifyPiTRSettings returns the pitr granularity/max age as strings to display in the report.
func (b *BucketBlueprint) stringifyPiTRSettings() (string, string) {
	if !b.PiTREnabled {
		return "N/A", "N/A"
	}

	pitrGranularity := "default"
	if b.PiTRGranularity != 0 {
		pitrGranularity = strconv.FormatUint(b.PiTRGranularity, 10)
	}

	pitrMaxHistoryAge := "default"
	if b.PiTRMaxHistoryAge != 0 {
		pitrMaxHistoryAge = strconv.FormatUint(b.PiTRMaxHistoryAge, 10)
	}

	return pitrGranularity, pitrMaxHistoryAge
}
