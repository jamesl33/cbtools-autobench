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
	"time"
)

// BenchmarkConfig encapsulates the configuration available for running benchmarks.
type BenchmarkConfig struct {
	// Iterations is the number of times a benchmark will be run, more iterations will result in more accurate data.
	Iterations int `json:"iterations,omitempty" yaml:"iterations,omitempty"`

	// CBMConfig is the configuration which will be passed to 'cbbackupmgr' when run on the remote machine.
	CBMConfig *CBMConfig `json:"cbbackupmgr_config,omitempty" yaml:"cbbackupmgr_config,omitempty"`
}

// BenchmarkResults is a wrapper around a slice of benchmark results which provides some utility functions.
type BenchmarkResults []*BenchmarkResult

// BenchmarkResult encapsulates a single benchmark results.
type BenchmarkResult struct {
	// Duration is the how long the benchmark took to complete (this does not include setup/cleanup).
	Duration time.Duration

	// ADS is the actual size of the data that was backed up. This will be used to calculate how much data is
	// transferred for backup/restore benchmarks.
	ADS uint64
}

// AvgTransferRateGDS returns the average transfer rate of all the benchmarks calculated using the generated data size.
func (b *BenchmarkResult) AvgTransferRateGDS(blueprint *DataBlueprint) uint64 {
	if b.Duration < time.Second {
		return uint64(blueprint.Size * blueprint.Items)
	}

	return uint64(blueprint.Size*blueprint.Items) / uint64(b.Duration.Seconds())
}

// AvgTransferRateADS returns the average transfer rate of all the benchmarks calculated using the actual data size.
func (b *BenchmarkResult) AvgTransferRateADS() uint64 {
	if b.Duration < time.Second {
		return b.ADS
	}

	return b.ADS / uint64(b.Duration.Seconds())
}
