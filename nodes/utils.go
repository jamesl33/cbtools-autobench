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

package nodes

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/jamesl33/cbtools-autobench/value"
)

// latestBuilds is the address which is used when downloading builds.
const latestBuilds = "latestbuilds.service.couchbase.com/builds/latestbuilds/couchbase-server"

// createBuildURL returns the address which should be used to download the provided build.
func createBuildURL(platform value.Platform, build string) (string, error) {
	match := regexp.MustCompile("^" + value.RegexBuildID + "$").FindStringSubmatch(build)
	if match == nil {
		return "", fmt.Errorf("unknown build version '%s'", build)
	}

	// The 'Join' function implictly calls 'Clean' on the returned path, therefore, we must prefix 'http://' to the
	// returned value.
	return "http://" + path.Join(
		latestBuilds,
		versionToCodename(match[1]),
		match[2],
		fmt.Sprintf("couchbase-server-enterprise_%s-%s_amd64.%s", build, platform, platform.PackageExtension()),
	), nil
}

// versionToCodename returns the codename for the provided version.
func versionToCodename(version string) string {
	if strings.HasPrefix(version, "7") {
		return "cheshire-cat"
	}

	if strings.HasPrefix(version, "6") {
		return "mad-hatter"
	}

	panic(fmt.Sprintf("unsupported version '%s'", version))
}
