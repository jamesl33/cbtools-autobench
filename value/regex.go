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

// RegexBuildID is an uncompiled regular expression which may be used to extract information from a Couchbase build
// identifier.
//
// Full match: 7.0.0-4259
// Group 1: 7.0.0
// Group 2: 4259
const RegexBuildID = `(\d+\.\d+\.\d+)-(\d+)`
