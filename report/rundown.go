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
	"strings"
	"text/tabwriter"

	"github.com/couchbase/tools-common/format"
)

// rundownResult encapsulates the information for a single benchmark iteration.
type rundownResult struct {
	Duration           string `json:"duration,omitempty"`
	AIN                string `json:"ain,omitempty"`
	ADS                string `json:"ads,omitempty"`
	GDS                string `json:"gds,omitempty"`
	AvgTransferRateADS string `json:"avg_transfer_rate_ads,omitempty"`
	AvgTransferRateGDS string `json:"avg_transfer_rate_gds,omitempty"`
}

// Rundown is a component which contains the detailed rundown for each benchmark that was executed.
type Rundown []*rundownResult

// NewRundown creates a new 'Rundown' component with the provided options.
func NewRundown(options Options) Rundown {
	results := make([]*rundownResult, 0, len(options.Results))
	for _, result := range options.Results {
		results = append(results, &rundownResult{
			Duration: format.Duration(result.Duration),
			AIN:      fmt.Sprint(result.AIN),
			ADS:      format.Bytes(result.ADS),
			GDS: format.Bytes(uint64(options.Blueprint.Cluster.Bucket.Data.Items *
				options.Blueprint.Cluster.Bucket.Data.Size)),
			AvgTransferRateADS: format.Bytes(result.AvgTransferRateADS()),
			AvgTransferRateGDS: format.Bytes(result.AvgTransferRateGDS(options.Blueprint.Cluster.Bucket.Data)),
		})
	}

	return results
}

// String returns a string representation of the 'Rundown' component which will be output in the report.
func (r Rundown) String() string {
	var (
		buffer = &bytes.Buffer{}
		writer = tabwriter.NewWriter(buffer, 4, 0, 1, ' ', tabwriter.Debug)
	)

	fmt.Fprintln(buffer, "| Rundown\n| -------")
	fmt.Fprintf(writer, "| Iteration\t Duration\t Items (AIN)\t Size (ADS)\t Size (GDS)\t Transfer Rate (ADS)\t "+
		"Transfer Rate (GDS)\t\n")

	for index, result := range r {
		fmt.Fprintf(writer, "| %d\t %s\t %s\t %s\t %s\t %s/s\t %s/s\t\n",
			index+1,
			result.Duration,
			result.AIN,
			result.ADS,
			result.GDS,
			result.AvgTransferRateADS,
			result.AvgTransferRateGDS)
	}

	_ = writer.Flush()

	return strings.TrimSpace(buffer.String())
}
