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
	"github.com/jamesl33/cbtools-autobench/value"
)

// Options encapsulates the options which may be passed into the 'NewReport' function and avoids having ungainly
// function signatures.
type Options struct {
	Blueprint   *value.Blueprint
	Stats       *value.Stats
	CBMConfig   *value.CBMConfig
	Results     value.BenchmarkResults
	ClusterLogs []string
	BackupLogs  string
}
