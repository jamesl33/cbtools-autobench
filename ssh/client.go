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

package ssh

import (
	"fmt"
	"net"

	fsutil "github.com/couchbase/tools-common/fs/util"
	"github.com/jamesl33/cbtools-autobench/value"

	"github.com/apex/log"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

// TODO (jamesl33) We really shouldn't be using 'os.TempDir' when running commands on remote machines since the
// temporary directory from the local machine might not be valid on the remote machine. For the time being we only
// support Linux so this shouldn't be a major issue.

// Client is thin wrapper around an ssh client which exposes some useful functionality required when setting
// up/performing benchmarks.
type Client struct {
	client   *ssh.Client
	Platform value.Platform
}

// NewClient creates a new client which is connected to the provided host.
func NewClient(host string, config *value.SSHConfig) (*Client, error) {
	log.WithField("host", host).Info("Establishing ssh connection")

	signer, err := parsePrivateKey(config.PrivateKey, config.PrivateKeyPassphrase)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse private key")
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, 22), &ssh.ClientConfig{
		User:            config.Username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: func(_ string, _ net.Addr, _ ssh.PublicKey) error { return nil },
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ssh client")
	}

	platform, err := determinePlatform(client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to determine platform")
	}

	fields := log.Fields{"platform": platform, "host": host}
	log.WithFields(fields).Info("Successfully established ssh connection")

	return &Client{
		Platform: platform,
		client:   client,
	}, nil
}

// SecureUpload emulates the 'scp' command by uploading the file at the provided path to the remote server.
func (c *Client) SecureUpload(source, sink string) error {
	fields := log.Fields{
		"local":  trimPort(c.client.LocalAddr().String()),
		"remote": trimPort(c.client.RemoteAddr().String()),
		"source": source,
		"sink":   sink,
	}

	log.WithFields(fields).Debug("Uploading file")

	session, err := c.client.NewSession()
	if err != nil {
		return errors.Wrap(err, "failed to create session")
	}
	defer session.Close()

	pipe, err := session.StdinPipe()
	if err != nil {
		return errors.Wrap(err, "failed to get stdin pipe")
	}

	err = session.Start(fmt.Sprintf("cat > %s", sink))
	if err != nil {
		return errors.Wrap(err, "failed to start session")
	}

	err = fsutil.CopyFileTo(source, pipe)
	if err != nil {
		return errors.Wrap(err, "failed to copy source data to pipe")
	}

	err = pipe.Close()
	if err != nil {
		return errors.Wrap(err, "failed to close pipe")
	}

	return session.Wait()
}

// SecureDownload emulates the 'scp' command by downloaded the file at the provided path to the local machine.
func (c *Client) SecureDownload(source, sink string) error {
	fields := log.Fields{
		"local":  trimPort(c.client.LocalAddr().String()),
		"remote": trimPort(c.client.RemoteAddr().String()),
		"source": source,
		"sink":   sink,
	}

	log.WithFields(fields).Debug("Downloading file")

	session, err := c.client.NewSession()
	if err != nil {
		return errors.Wrap(err, "failed to create session")
	}
	defer session.Close()

	pipe, err := session.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "failed to get stdout pipe")
	}

	err = session.Start(fmt.Sprintf("cat %s", source))
	if err != nil {
		return errors.Wrap(err, "failed to start session")
	}

	err = fsutil.WriteToFile(sink, pipe, 0)
	if err != nil {
		return errors.Wrap(err, "failed to copy to file")
	}

	return session.Wait()
}

// InstallPackageAt installs the package at the provided path on the remote machine.
func (c *Client) InstallPackageAt(path string) error {
	_, err := c.ExecuteCommand(c.Platform.CommandInstallPackageAt(path))
	return err
}

// InstallPackages uses the platform specific package manager to install the given package.
func (c *Client) InstallPackages(packages ...string) error {
	_, err := c.ExecuteCommand(c.Platform.CommandInstallPackages(packages...))
	return err
}

// UninstallPackages uses the platform specific package manager to uninstall the given package.
func (c *Client) UninstallPackages(packages ...string) error {
	_, err := c.ExecuteCommand(c.Platform.CommandUninstallPackages(packages...))
	return err
}

// FileExists returns a boolean indicating whether a file with the given path exists on the remote machine.
func (c *Client) FileExists(path string) bool {
	_, err := c.ExecuteCommand(value.NewCommand("test -e %s", path))
	return err == nil
}

// RemoveFile removes the file at the given path on the remote machine.
func (c *Client) RemoveFile(path string) error {
	_, err := c.ExecuteCommand(value.NewCommand("rm %s", path))
	return err
}

// RemoveDirectory removes the directory at the given path on the remote machine.
func (c *Client) RemoveDirectory(path string) error {
	_, err := c.ExecuteCommand(value.NewCommand("rm -rf %s", path))
	return err
}

// Sync runs 'sync' on the remote machine ensuring all dirty package are written to disk.
func (c *Client) Sync() error {
	_, err := c.ExecuteCommand(value.NewCommand("sync"))
	return err
}

// FlushCaches sync then flushes the caches on the remote machine; this allows for more consistent benchmark results.
func (c *Client) FlushCaches() error {
	_, err := c.ExecuteCommand(value.NewCommand("sync; echo 3 > /proc/sys/vm/drop_caches"))
	return err
}

// ExecuteCommand is a wrapper with executes the given command on the remote machine.
func (c *Client) ExecuteCommand(command value.Command) ([]byte, error) {
	return executeCommand(c.client, command.ToString(map[string]string{
		"PATH": fmt.Sprintf("%s:$PATH", value.CBBinDirectory),
	}))
}

// Close releases an resources in use by this client.
func (c *Client) Close() error {
	return c.client.Close()
}
