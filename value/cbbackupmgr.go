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
	"strings"
	"text/tabwriter"
)

// CBMEnvironment is the environment that will be passed to 'cbbackupmgr' when it's run on the remote machine.
type CBMEnvironment map[string]string

func (c CBMEnvironment) String() string {
	var (
		buffer = &bytes.Buffer{}
		writer = tabwriter.NewWriter(buffer, 4, 0, 1, ' ', tabwriter.Debug)
	)

	fmt.Fprintln(buffer, "| CBM Environment Variables\n| -------------------------")
	fmt.Fprintf(writer, "| Key\t Value\t\n")

	for key, value := range c {
		fmt.Fprintf(writer, "| %s\t %s\t\n", key, value)
	}

	_ = writer.Flush()

	return buffer.String()
}

// CBMConfig encapsulates the available config for 'cbbackupmgr' and is used when commands are run on the remote
// machine.
type CBMConfig struct {
	// EnvVars is the environment that will be passed to 'cbbackupmgr'.
	//
	// TODO (jamesl33) It would be nice if this was part of the output report.
	EnvVars CBMEnvironment `json:"environment_variables,omitempty" yaml:"environment_variables,omitempty"`

	// Archive/repository that 'cbbackupmgr' will use.
	Archive    string `json:"archive,omitempty" yaml:"archive,omitempty"`
	Repository string `json:"repository,omitempty" yaml:"repository,omitempty"`

	// Storage is the storage type that will be used. This is a hidden command in 'cbbackupmgr' and is unsupported.
	Storage string `json:"storage,omitempty" yaml:"storage,omitempty"`

	// Cloud related arguments.
	ObjStagingDirectory       string `json:"obj_staging_directory,omitempty" yaml:"obj_staging_directory,omitempty"`
	ObjAccessKeyID            string `json:"-" yaml:"obj_access_key_id,omitempty"`
	ObjSecretAccessKey        string `json:"-" yaml:"obj_secret_access_key,omitempty"`
	ObjRegion                 string `json:"obj_region,omitempty" yaml:"obj_region,omitempty"`
	ObjEndpoint               string `json:"obj_endpoint,omitempty" yaml:"obj_endpoint,omitempty"`
	ObjAuthByInstanceMetadata bool   `json:"obj_auth_by_instance_metadata,omitempty" yaml:"obj_auth_by_instance_metadata,omitempty"` //nolint:lll
	ObjNoSSLVerify            bool   `json:"obj_no_ssl_verify,omitempty" yaml:"obj_no_ssl_verify,omitempty"`
	S3LogLevel                string `json:"s3_log_level,omitempty" yaml:"s3_log_level,omitempty"`
	S3ForcePathStyle          bool   `json:"s3_force_path_style,omitempty" yaml:"s3_force_path_style,omitempty"`

	// Encrypted related arguments
	Encrypted      bool   `json:"encrypted,omitempty" yaml:"encrypted,omitempty"`
	Passphrase     string `json:"passphrase,omitempty" yaml:"passphrase,omitempty"`
	EncryptionAlgo string `json:"encryption_algo,omitempty" yaml:"encryption_algo,omitempty"`

	// NumThreads is the default of threads which will be used by 'cbbackupmgr'. A zero value will allow 'cbbackupmgr'
	// to automatically determine the number of threads.
	Threads int `json:"threads,omitempty" yaml:"threads,omitempty"`

	// Blackhole indicates whether the benchmarks should actually backup any data or just pull it from the cluster and
	// then discard it immediately.
	Blackhole bool `json:"blackhole,omitempty" yaml:"blackhole,omitempty"`
}

// String returns a human readable string representation of the config which will be displayed in the report.
func (c *CBMConfig) String() string {
	var (
		buffer = &bytes.Buffer{}
		writer = tabwriter.NewWriter(buffer, 4, 0, 1, ' ', tabwriter.Debug)
	)

	staging := "N/A"
	if c.ObjStagingDirectory != "" {
		staging = c.ObjStagingDirectory
	}

	storage := "default"
	if c.Storage != "" {
		storage = c.Storage
	}

	threads := "auto"
	if c.Threads != 0 {
		threads = strconv.Itoa(c.Threads)
	}

	fmt.Fprintln(buffer, "| CBM\n| ----")
	fmt.Fprintf(writer, "| Archive\t Repository \t Staging Directory\t Storage\t Threads\t Blackhole\t\n")
	fmt.Fprintf(writer, "| %s\t %s\t %s\t %s\t %s\t %t\t\n",
		c.Archive,
		c.Repository,
		staging,
		storage,
		threads,
		c.Blackhole)

	_ = writer.Flush()

	if len(c.EnvVars) != 0 {
		fmt.Fprintf(buffer, "\n%s", c.EnvVars)
	}

	return strings.TrimSpace(buffer.String())
}

// CommandConfig returns a command which may be run on the remote backup client to configure the benchmark
// archive/repository.
func (c *CBMConfig) CommandConfig() Command {
	command := fmt.Sprintf(`cbbackupmgr config -a %s -r %s`, c.Archive, c.Repository)

	command = c.prefixEnvironment(command)
	command = c.addCloudArgs(command)
	command = c.addEncryptionArgs(command, true)

	return NewCommand(command)
}

// CommandBackup returns a command which may be run on the remote backup client to perform a backup.
func (c *CBMConfig) CommandBackup(host string, ignoreBlackhole bool) Command {
	command := fmt.Sprintf(
		`cbbackupmgr backup -a %s -r %s -c %s -u Administrator -p asdasd --no-progress-bar`,
		c.Archive,
		c.Repository,
		host,
	)

	command = c.prefixEnvironment(command)
	command = c.addCloudArgs(command)
	command = c.addEncryptionArgs(command, false)
	command = c.addStorage(command)
	command = c.addThreads(command)

	// When we're performing restore benchmarks we actually need to create a backup so we should ignore the blackhole
	// configuration.
	if !ignoreBlackhole {
		command = c.addBlackhole(command)
	}

	return NewCommand(command)
}

// CommandRestore returns a command which can be run on the remote backup client to perform a restore.
func (c *CBMConfig) CommandRestore(host string) Command {
	command := fmt.Sprintf(
		`cbbackupmgr restore -a %s -r %s -c %s -u Administrator -p asdasd --no-progress-bar`,
		c.Archive,
		c.Repository,
		host,
	)

	command = c.prefixEnvironment(command)
	command = c.addCloudArgs(command)
	command = c.addEncryptionArgs(command, false)
	command = c.addThreads(command)
	command = c.addBlackhole(command)

	return NewCommand(command)
}

// CommandCollectLogs returns a command which can be run on the remote backup client to collect the 'cbbackupmgr' logs.
func (c *CBMConfig) CommandCollectLogs() Command {
	command := fmt.Sprintf(`cbbackupmgr collect-logs -a %s`, c.Archive)

	command = c.addCloudArgs(command)
	command = c.prefixEnvironment(command)

	return NewCommand(command)
}

// CommandRemove returns a command which can be run on the remote backup client to remove all the backups from start to
// end.
func (c *CBMConfig) CommandRemove(start, end string) Command {
	command := fmt.Sprintf(
		"cbbackupmgr remove -a %s -r %s --backups %s,%s",
		c.Archive,
		c.Repository,
		start,
		end,
	)

	command = c.prefixEnvironment(command)
	command = c.addCloudArgs(command)

	return NewCommand(command)
}

// CommandInfo returns a command which can be run on the remote backup client which will return information about the
// given backup repository in JSON format.
func (c *CBMConfig) CommandInfo() Command {
	command := fmt.Sprintf("cbbackupmgr info -a %s -r %s -j", c.Archive, c.Repository)

	command = c.prefixEnvironment(command)
	command = c.addCloudArgs(command)

	return NewCommand(command)
}

// prefixEnvironment with prefix the given command with the current 'cbbackupmgr' environment variables.
func (c *CBMConfig) prefixEnvironment(command string) string {
	if len(c.EnvVars) == 0 {
		return command
	}

	var env string
	for key, value := range c.EnvVars {
		env += fmt.Sprintf("export %s=%s; ", key, value)
	}

	return env + command
}

// addStorage will add the storage flag to the given command if required.
func (c *CBMConfig) addStorage(command string) string {
	if c.Storage == "" {
		return command
	}

	return command + fmt.Sprintf(" --storage %s", c.Storage)
}

// addThreads will add the --threads/--auto-select-threads flag to the given command.
func (c *CBMConfig) addThreads(command string) string {
	if c.Threads != 0 {
		return command + fmt.Sprintf(" --threads %d", c.Threads)
	}

	return command + " --auto-select-threads"
}

// addBlackhole will conditionally add the --blackhole flag to the given command.
func (c *CBMConfig) addBlackhole(command string) string {
	if !c.Blackhole {
		return command
	}

	return command + " --sink blackhole"
}

// addCloudArgs will conditionally add the provided cloud flags to the given command.
func (c *CBMConfig) addCloudArgs(command string) string {
	if c.ObjStagingDirectory != "" {
		command += fmt.Sprintf(" --obj-staging-dir %s", c.ObjStagingDirectory)
	}

	if c.ObjAccessKeyID != "" {
		command += fmt.Sprintf(" --obj-access-key-id %s", c.ObjAccessKeyID)
	}

	if c.ObjSecretAccessKey != "" {
		command += fmt.Sprintf(" --obj-secret-access-key %s", c.ObjSecretAccessKey)
	}

	if c.ObjRegion != "" {
		command += fmt.Sprintf(" --obj-region %s", c.ObjRegion)
	}

	if c.ObjEndpoint != "" {
		command += fmt.Sprintf(" --obj-endpoint %s", c.ObjEndpoint)
	}

	if c.ObjAuthByInstanceMetadata {
		command += " --obj-auth-by-instance-metadata"
	}

	if c.ObjNoSSLVerify {
		command += " --obj-no-ssl-verify"
	}

	if c.S3LogLevel != "" {
		command += fmt.Sprintf(" --s3-log-level %s", c.S3LogLevel)
	}

	if c.S3ForcePathStyle {
		command += " --s3-force-path-style"
	}

	return command
}

// addEncryptionArgs will conditionally add the provided encryption flags to the given command.
func (c *CBMConfig) addEncryptionArgs(command string, config bool) string {
	if !c.Encrypted {
		return command
	}

	command += fmt.Sprintf(" --passphrase %s", c.Passphrase)

	if !config {
		return command
	}

	command += " --encrypted"

	if c.EncryptionAlgo != "" {
		command += fmt.Sprintf(" --encryption-algo %s", c.EncryptionAlgo)
	}

	return command
}
