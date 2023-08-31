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
	"fmt"
	"strings"
)

// Platform represents the platform that 'cbtools-autobench' is currently being run against (note this is referring to
// the remote machine).
//
// NOTE: At the moment, only Linux is supported, however, package managers and package names may differ; this means
// additional work may be required to handle different distributions.
type Platform string

const (
	// PlatformUbuntu20_04 represents the 20.04 release of Ubuntu.
	PlatformUbuntu20_04 Platform = "ubuntu20.04"

	// PlatformAmazonLinux2 represents the second version of Amazon Linux, note that the first version is now hidden
	// from users and in theory should no longer be used.
	PlatformAmazonLinux2 Platform = "amzn2"
)

// PackageExtension returns the extension used by this platforms package manager.
func (p Platform) PackageExtension() string {
	switch p {
	case PlatformUbuntu20_04:
		return "deb"
	case PlatformAmazonLinux2:
		return "rpm"
	}

	panic(fmt.Sprintf("unsupported platform '%s'", p))
}

// Dependencies returns a list of package names which will be installed if they are missing.
func (p Platform) Dependencies() []string {
	switch p {
	case PlatformUbuntu20_04:
		return []string{"awscli", "libtinfo5"}
	case PlatformAmazonLinux2:
		return []string{"awscli", "ncurses-compat-libs"}
	}

	panic(fmt.Sprintf("unsupported platform '%s'", p))
}

// CommandInstallPackageAt returns a command which can be used to install the package at the provided path.
func (p Platform) CommandInstallPackageAt(path string) Command {
	switch p {
	case PlatformUbuntu20_04:
		return NewCommand("dpkg -i %s", path)
	case PlatformAmazonLinux2:
		return NewCommand("yum install -y %s", path)
	}

	panic(fmt.Sprintf("unsupported platform '%s'", p))
}

// CommandInstallPackages returns a command which can be used to installed the provided list of packages by name.
func (p Platform) CommandInstallPackages(packages ...string) Command {
	switch p {
	case PlatformUbuntu20_04:
		return NewCommand("apt update && apt install -y %s", strings.Join(packages, " "))
	case PlatformAmazonLinux2:
		return NewCommand("yum update -y && yum install -y %s", strings.Join(packages, " "))
	}

	panic(fmt.Sprintf("unsupported platform '%s'", p))
}

// CommandUninstallPackages returns a command which can be used to uninstall the provided list of package by name.
func (p Platform) CommandUninstallPackages(packages ...string) Command {
	switch p {
	case PlatformUbuntu20_04:
		return NewCommand("dpkg --purge %s", strings.Join(packages, " "))
	case PlatformAmazonLinux2:
		return NewCommand("yum autoremove -y %s", strings.Join(packages, " "))
	}

	panic(fmt.Sprintf("unsupported platform '%s'", p))
}

// CommandDisableCouchbase returns a command which when executed on the remote machine will disable Couchbase Server.
func (p Platform) CommandDisableCouchbase() Command {
	switch p {
	case PlatformUbuntu20_04, PlatformAmazonLinux2:
		return NewCommand("systemctl disable --now couchbase-server")
	}

	panic(fmt.Sprintf("unsupported platform '%s'", p))
}
