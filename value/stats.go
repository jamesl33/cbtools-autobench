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

	"github.com/couchbase/tools-common/strings/format"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Stats encapsulates some useful stats from the current cluster which may be displayed in the report to give more
// context about the benchmark conditions.
type Stats struct {
	ItemCount              uint64 `json:"itemCount"`
	DiskUsed               uint64 `json:"diskUsed"`
	MemUsed                uint64 `json:"memUsed"`
	VBActiveNumNonResident uint64 `json:"vbActiveNumNonResident"`
}

// MarshalJSON returns a JSON representation of the stats with raw values converted into human readable strings.
func (b *Stats) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ItemCount      uint64 `json:"item_count,omitempty"`
		MemoryUsed     string `json:"memory_used,omitempty"`
		DiskUsed       string `json:"disk_used,omitempty"`
		ResidencyRatio uint64 `json:"residency_ratio,omitempty"`
	}{
		ItemCount:      b.ItemCount,
		MemoryUsed:     format.Bytes(b.MemUsed),
		DiskUsed:       format.Bytes(b.DiskUsed),
		ResidencyRatio: residencyRatio(b.ItemCount, b.VBActiveNumNonResident),
	})
}

// String returns a string representation of the blueprint which will be output in the report.
func (b *Stats) String() string {
	var (
		buffer = &bytes.Buffer{}
		writer = tabwriter.NewWriter(buffer, 4, 0, 1, ' ', tabwriter.Debug)
	)

	fmt.Fprintln(buffer, "| Stats\n| -----")
	fmt.Fprintf(writer, "| Item Count\t Memory Used\t Disk Used\t Residency Ratio\t\n")
	fmt.Fprintf(writer, "| %s\t %s\t %s\t %d%%\t\n",
		message.NewPrinter(language.English).Sprintf("%d", b.ItemCount),
		format.Bytes(b.MemUsed),
		format.Bytes(b.DiskUsed),
		residencyRatio(b.ItemCount, b.VBActiveNumNonResident))

	_ = writer.Flush()

	return strings.TrimSpace(buffer.String())
}

// residencyRatio returns the current residency ratio using the same method as in the Couchbase Server WebUI.
func residencyRatio(items, nonResident uint64) uint64 {
	if items == 0 {
		return 100
	}

	if items < nonResident {
		return 0
	}

	return ((items - nonResident) * 100) / items
}
