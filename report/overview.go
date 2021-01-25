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
	"time"

	"github.com/couchbase/tools-common/format"
)

// Overview is the overview component to the report which highlights interesting averages across the benchmark
// iterations.
type Overview struct {
	AvgDuration        string `json:"avg_duration,omitempty"`
	AvgADS             string `json:"avg_ads,omitempty"`
	AvgGDS             string `json:"avg_gds,omitempty"`
	AvgTransferRateADS string `json:"avg_transfer_rate_ads,omitempty"`
	AvgTransferRateGDS string `json:"avg_transfer_rate_gds,omitempty"`
}

// NewOverview creates a new overview component with the provided options.
func NewOverview(options Options) *Overview {
	var (
		duration        time.Duration
		ads             uint64
		gds             uint64
		transferRateADS uint64
		transferRateGDS uint64
	)

	for _, result := range options.Results {
		duration += result.Duration
		ads += result.ADS
		gds += uint64(options.Blueprint.Cluster.Bucket.Data.Items * options.Blueprint.Cluster.Bucket.Data.Size)
		transferRateADS += result.AvgTransferRateADS()
		transferRateGDS += result.AvgTransferRateGDS(options.Blueprint.Cluster.Bucket.Data)
	}

	return &Overview{
		AvgDuration:        format.Duration(time.Duration(int64(duration) / int64(len(options.Results)))),
		AvgADS:             format.Bytes(ads / uint64(len(options.Results))),
		AvgGDS:             format.Bytes(gds / uint64(len(options.Results))),
		AvgTransferRateADS: format.Bytes(transferRateADS / uint64(len(options.Results))),
		AvgTransferRateGDS: format.Bytes(transferRateGDS / uint64(len(options.Results))),
	}
}

// String returns a string representation of the 'Logs' component which will be output in the report.
func (o *Overview) String() string {
	var (
		buffer = &bytes.Buffer{}
		writer = tabwriter.NewWriter(buffer, 4, 0, 1, ' ', tabwriter.Debug)
	)

	fmt.Fprintln(buffer, "| Overview\n| --------")
	fmt.Fprintf(writer,
		"| Avg Duration\t Avg Size (ADS)\t Avg Size (GDS)\t Avg Transfer Rate (ADS)\t Avg Transfer Rate (GDS)\t\n")
	fmt.Fprintf(writer, "| %s\t %s\t %s\t %s/s\t %s/s\t\n",
		o.AvgDuration,
		o.AvgADS,
		o.AvgGDS,
		o.AvgTransferRateADS,
		o.AvgTransferRateGDS)

	_ = writer.Flush()

	return strings.TrimSpace(buffer.String())
}
