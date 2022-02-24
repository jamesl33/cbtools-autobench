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

	"github.com/couchbase/tools-common/format"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type DataLoaderType string

const (
	CBM         DataLoaderType = "cbbackupmgr"
	Pillowfight DataLoaderType = "pillowfight"
)

// DataBlueprint encapsulates all the options available when populating a bucket with benchmarking data.
type DataBlueprint struct {
	DataLoader   DataLoaderType `json:"data_loader,omitempty" yaml:"data_loader,omitempty"`
	Items        int            `json:"items,omitempty" yaml:"items,omitempty"`
	ActiveItems  int            `json:"active_items,omitempty" yaml:"active_items,omitempty"`
	Size         int            `json:"size,omitempty" yaml:"size,omitempty"`
	Compressible bool           `json:"compressible,omitempty" yaml:"compressible,omitempty"`
	LoadThreads  int            `json:"load_threads,omitempty" yaml:"load_threads,omitempty"`
}

// String returns a string representation of the blueprint which will be output in the report.
func (d *DataBlueprint) String() string {
	var (
		buffer = &bytes.Buffer{}
		writer = tabwriter.NewWriter(buffer, 4, 0, 1, ' ', tabwriter.Debug)
	)

	threads := "auto"
	if d.LoadThreads != 0 {
		threads = strconv.Itoa(d.LoadThreads)
	}

	activeItems := "N/A"
	if d.DataLoader == Pillowfight && d.ActiveItems != 0 {
		activeItems = message.NewPrinter(language.English).Sprintf("%d", d.ActiveItems)
	}

	fmt.Fprintln(buffer, "| Data\n| ----")
	fmt.Fprintf(writer, "| Data Loader\t Items\t Active Items\t Size\t Compressible\t Load Threads\t\n")
	fmt.Fprintf(writer, "| %s\t %s\t %s\t %s\t %t\t %s\t\n",
		d.DataLoader,
		message.NewPrinter(language.English).Sprintf("%d", d.Items),
		activeItems,
		format.Bytes(uint64(d.Size)),
		d.Compressible,
		threads)

	_ = writer.Flush()

	return buffer.String()
}
