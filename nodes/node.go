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
	"os"
	"path/filepath"
	"time"

	"github.com/jamesl33/cbtools-autobench/ssh"
	"github.com/jamesl33/cbtools-autobench/value"

	"github.com/apex/log"
	"github.com/pkg/errors"
)

// Node represents a connection to a remote Couchbase Server node (note that the node may or may not be setup yet).
type Node struct {
	blueprint *value.NodeBlueprint
	client    *ssh.Client
}

// NewNode creates a connection to the remote node using the provided ssh config.
func NewNode(config *value.SSHConfig, blueprint *value.NodeBlueprint) (*Node, error) {
	client, err := ssh.NewClient(blueprint.Host, config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ssh client")
	}

	return &Node{blueprint: blueprint, client: client}, nil
}

// provision the node by installing the required dependencies (including Couchbase Server).
func (n *Node) provision(path string) error {
	err := n.installDeps()
	if err != nil {
		return errors.Wrap(err, "failed to install dependencies")
	}

	err = n.uninstallCB()
	if err != nil {
		return errors.Wrap(err, "failed to uninstall Couchbase Server")
	}

	err = n.installCB(path)
	if err != nil {
		return errors.Wrap(err, "failed to install Couchbase Server")
	}

	// We've got to wait for things to complete, for example we need to actually wait for Couchbase Server to start
	time.Sleep(30 * time.Second)

	return nil
}

// installDeps installs any required platform specific dependencies which are missing on the remote machine.
func (n *Node) installDeps() error {
	log.WithField("host", n.blueprint.Host).Info("Installing dependencies")

	return n.client.InstallPackages(n.client.Platform.Dependencies()...)
}

// uninstallCB will uninstall Couchbase Server from the remote node ensuring a clean slate.
func (n *Node) uninstallCB() error {
	log.WithField("host", n.blueprint.Host).Info("Uninstalling 'couchbase-server'")

	err := n.client.UninstallPackages("couchbase-server")
	if err != nil {
		return errors.Wrap(err, "failed to uninstall 'couchbase-server'")
	}

	log.WithField("host", n.blueprint.Host).Info("Purging install directory")

	err = n.client.RemoveDirectory(value.CBInstallDirectory)
	if err != nil {
		return errors.Wrapf(err, "failed to cleanup install directory at '%s'", value.CBInstallDirectory)
	}

	return nil
}

// installCB uploads the Couchbase Server install package to the remote machine and installs it.
//
// NOTE: The package archive will be removed upon completion.
func (n *Node) installCB(localPath string) error {
	remotePath := filepath.Join(os.TempDir(), filepath.Base(localPath))

	log.WithField("host", n.blueprint.Host).Info("Uploading package archive")

	err := n.client.SecureUpload(localPath, remotePath)
	if err != nil {
		return errors.Wrap(err, "failed to upload package archive")
	}

	log.WithField("host", n.blueprint.Host).Info("Installing 'couchbase-server'")

	err = n.client.InstallPackageAt(remotePath)
	if err != nil {
		return errors.Wrap(err, "failed to install 'couchbase-server'")
	}

	log.WithField("host", n.blueprint.Host).Info("Cleaning up package archive")

	err = n.client.RemoveFile(remotePath)
	if err != nil {
		return errors.Wrap(err, "failed to remove package archive")
	}

	return nil
}

// createDataPath ensures that the users chosen data path exists on the remote machine.
func (n *Node) createDataPath() error {
	if n.blueprint.DataPath == "" {
		return nil
	}

	log.WithField("host", n.blueprint.Host).Info("Creating/configuring data path")

	_, err := n.client.ExecuteCommand(value.NewCommand("mkdir -p %s", n.blueprint.DataPath))
	if err != nil {
		return errors.Wrap(err, "failed to create remote data directory")
	}

	_, err = n.client.ExecuteCommand(value.NewCommand("chown -R couchbase:couchbase %s", n.blueprint.DataPath))
	if err != nil {
		return errors.Wrap(err, "failed to chown remote data directory")
	}

	return nil
}

// initializeCB will perform node level initialization of Couchbase Server.
func (n *Node) initializeCB() error {
	fields := log.Fields{"host": n.blueprint.Host, "data_path": n.blueprint.DataPath}
	log.WithFields(fields).Info("Initializing node")

	init := "couchbase-cli node-init -c localhost:8091 -u Administrator -p asdasd"
	if n.blueprint.DataPath != "" {
		init += fmt.Sprintf(" --node-init-data-path %s", n.blueprint.DataPath)
	}

	_, err := n.client.ExecuteCommand(value.NewCommand(init))

	return err
}

// disableCB will disable Couchbase Server on the remote node, this will done on the backup client to free up resources
// for 'cbbackupmgr'.
func (n *Node) disableCB() error {
	log.WithField("host", n.blueprint.Host).Info("Disabling 'couchbase-server'")

	_, err := n.client.ExecuteCommand(n.client.Platform.CommandDisableCouchbase())

	return err
}

// Close releases any resources in use by the connection.
func (n *Node) Close() error {
	return n.client.Close()
}
