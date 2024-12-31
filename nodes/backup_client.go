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
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/jamesl33/cbtools-autobench/value"

	"github.com/apex/log"
	"github.com/pkg/errors"
)

// BackupClient represents a connection to a backup client/node and can be used to perform provisioning/benchmarking.
type BackupClient struct {
	blueprint *value.BackupClientBlueprint
	node      *Node
}

// NewBackupClient will connect to a backup client using the provided config.
func NewBackupClient(config *value.SSHConfig, blueprint *value.BackupClientBlueprint) (*BackupClient, error) {
	node, err := NewNode(config, &value.NodeBlueprint{Host: blueprint.Host})
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to node")
	}

	return &BackupClient{
		blueprint: blueprint,
		node:      node,
	}, nil
}

// Provision will use the client blueprint to provision the backup client, note that if the client is already
// provisioned it will be re-provisioned i.e. we will remove then install Couchbase.
func (b *BackupClient) Provision() error {
	log.WithField("host", b.blueprint.Host).Info("Provisioning backup client")

	err := b.node.provision(b.blueprint.PackagePath)
	if err != nil {
		return errors.Wrap(err, "failed to provision node")
	}

	// The backup client doesn't need to be running Couchbase in the background, we should disable it so it's not
	// consuming any resources.
	err = b.node.disableCB()
	if err != nil {
		return errors.Wrap(err, "failed to disable Couchbase Server")
	}

	return nil
}

// CollectLogs will run 'collect-logs' on the backup client then cp/download the logs into the provided directory.
func (b *BackupClient) CollectLogs(config *value.BenchmarkConfig, path string) (string, error) {
	log.WithField("path", path).Info("Collecting 'cbbackupmgr' logs")

	_, err := b.node.client.ExecuteCommand(config.CBMConfig.CommandCollectLogs())
	if err != nil {
		return "", errors.Wrap(err, "failed to run 'collect-logs'")
	}

	local := config.CBMConfig.Archive
	if config.CBMConfig.ObjStagingDirectory != "" {
		local = config.CBMConfig.ObjStagingDirectory
	}

	output, err := b.node.client.ExecuteCommand(
		value.NewCommand(`ls -t %s | head -1`, filepath.Join(local, "logs", "*.zip")))
	if err != nil {
		return "", errors.Wrap(err, "failed to determine which zip file to cp/download")
	}

	var (
		source = strings.TrimSpace(string(output))
		sink   = filepath.Join(path, filepath.Base(source))
	)

	fields := log.Fields{"source": source, "sink": sink}
	log.WithFields(fields).Info("Downloading 'cbbackupmgr' logs")

	err = b.node.client.SecureDownload(source, sink)
	if err != nil {
		return "", errors.Wrap(err, "failed to cp/download logs")
	}

	return sink, nil
}

// BenchmarkBackup will run one or more backup benchmarks on the client using the provided benchmark config. If the
// provided context is cancelled, we will gracefully complete the current backup then return early.
func (b *BackupClient) BenchmarkBackup(ctx context.Context, config *value.BenchmarkConfig,
	cluster *Cluster,
) (value.BenchmarkResults, error) {
	log.WithField("iterations", config.Iterations).Info("Beginning 'cbbackupmgr' backup benchmark(s)")

	err := b.purgeArchive(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to purge archive")
	}

	err = b.createRepository(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create repository")
	}

	results := make(value.BenchmarkResults, 0, config.Iterations)

	for iteration := 0; iteration < max(1, config.Iterations); iteration++ {
		log.WithField("iteration", iteration+1).Info("Beginning 'cbbackupmgr' backup benchmark")

		result, err := b.benchmarkBackup(config, cluster)
		if err != nil {
			return nil, errors.Wrap(err, "failed to run benchmark")
		}

		results = append(results, result)

		// If the context has been cancelled, don't run any more benchmarks; the user wants to gracefully terminate
		if ctx.Err() != nil {
			break
		}
	}

	return results, nil
}

// BenchmarkRestore will run one or more restore benchmarks on the client using the providing benchmark config. If the
// provided context is cancelled, we will gracefully complete the current restore then return early.
func (b *BackupClient) BenchmarkRestore(ctx context.Context, config *value.BenchmarkConfig,
	cluster *Cluster,
) (value.BenchmarkResults, error) {
	log.WithField("iterations", config.Iterations).Info("Beginning 'cbbackupmgr' restore benchmark(s)")

	err := b.purgeArchive(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to purge archive")
	}

	err = b.createRepository(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create repository")
	}

	backupInfo, err := b.createBackup(config, cluster, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create backup")
	}

	results := make(value.BenchmarkResults, 0, config.Iterations)

	for iteration := 0; iteration < max(1, config.Iterations); iteration++ {
		log.WithField("iteration", iteration+1).Info("Beginning 'cbbackupmgr' restore benchmark")

		if !config.CBMConfig.Blackhole {
			err = cluster.flushBucket()
			if err != nil {
				return nil, errors.Wrap(err, "failed to flush bucket")
			}
		}

		result, err := b.benchmarkRestore(config, cluster, backupInfo.BackupSize)
		if err != nil {
			return nil, errors.Wrap(err, "failed to run benchmark")
		}

		results = append(results, result)

		// If the context has been cancelled, don't run any more benchmarks; the user wants to gracefully terminate
		if ctx.Err() != nil {
			break
		}
	}

	return results, nil
}

// benchmarkBackup will run an individual backup benchmark and fetch any data needed to produce a useful report.
func (b *BackupClient) benchmarkBackup(config *value.BenchmarkConfig,
	cluster *Cluster,
) (*value.BenchmarkResult, error) {
	result := &value.BenchmarkResult{}

	start := time.Now()
	defer func() {
		result.Duration = time.Since(start)
	}()

	err := cluster.runPreBenchmarkTasks()
	if err != nil {
		return nil, errors.Wrap(err, "failed to run cluster pre-benchmark tasks")
	}

	err = b.runPreBenchmarkTasks()
	if err != nil {
		return nil, errors.Wrap(err, "failed to run client pre-benchmark tasks")
	}

	backupInfo, err := b.createBackup(config, cluster, false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create backup")
	}

	result.ADS = backupInfo.BackupSize
	result.AIN = backupInfo.ItemsNum

	err = b.purgeBackups(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to purge created backup")
	}

	return result, nil
}

// benchmarkRestore will run an individual restore benchmark and fetch any data needed to produce a useful report.
func (b *BackupClient) benchmarkRestore(config *value.BenchmarkConfig,
	cluster *Cluster, ads uint64,
) (*value.BenchmarkResult, error) {
	result := &value.BenchmarkResult{
		ADS: ads,
	}

	start := time.Now()
	defer func() {
		result.Duration = time.Since(start)
	}()

	err := cluster.runPreBenchmarkTasks()
	if err != nil {
		return nil, errors.Wrap(err, "failed to run cluster pre-benchmark tasks")
	}

	err = b.runPreBenchmarkTasks()
	if err != nil {
		return nil, errors.Wrap(err, "failed to run client pre-benchmark tasks")
	}

	err = b.restoreBackup(config, cluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to restore backup")
	}

	return result, nil
}

// configureRepository wil run the config sub-command to create a new backup repository.
func (b *BackupClient) createRepository(config *value.BenchmarkConfig) error {
	log.Info("Creating repository")

	_, err := b.node.client.ExecuteCommand(config.CBMConfig.CommandConfig())

	return err
}

// runPreBenchmarkTasks will run any pre-benchmark tasks on the backup client. For example, we should always flush the
// caches prior to running a benchmark.
func (b *BackupClient) runPreBenchmarkTasks() error {
	log.Info("Running backup client pre-benchmark tasks")

	return b.node.client.FlushCaches()
}

// createBackup creates a backup of the provided cluster, note that the 'ignoreBlackhole' argument is required to allow
// benchmarking restore to blackhole i.e. we must create a backup to restore.
func (b *BackupClient) createBackup(config *value.BenchmarkConfig, cluster *Cluster,
	ignoreBlackhole bool,
) (*value.BackupInfo, error) {
	fields := log.Fields{
		"blackhole": config.CBMConfig.Blackhole,
		"hosts":     cluster.hosts(),
	}

	log.WithFields(fields).Info("Creating backup")

	command := config.CBMConfig.CommandBackup(cluster.ConnectionString(config.CBMConfig.TLS), ignoreBlackhole)

	_, err := b.node.client.ExecuteCommand(command)
	if err != nil {
		return nil, errors.Wrap(err, "failed to run backup")
	}

	// All the data should be synced to disk by cbbackupmgr, however, for good measure we'll sync now
	err = b.node.client.Sync()
	if err != nil {
		return nil, errors.Wrap(err, "failed to sync data to disk")
	}

	output, err := b.node.client.ExecuteCommand(config.CBMConfig.CommandInfo())
	if err != nil {
		return nil, errors.Wrap(err, "failed to run info")
	}

	type overlayBucket struct {
		Items uint64 `json:"total_mutations"`
	}

	type overlayBackup struct {
		Size    uint64          `json:"size"`
		Buckets []overlayBucket `json:"buckets"`
	}

	type overlay struct {
		Backups []overlayBackup `json:"backups"`
	}

	var decoded overlay

	err = json.Unmarshal(output, &decoded)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode info output")
	}

	backupInfo := &value.BackupInfo{
		// On each iteration we only do one backup so we only care about the size of the first and only backup in the
		// list
		BackupSize: decoded.Backups[0].Size,
		// We are only backing up one bucket so we can get the number of items from the first and only bucket
		// NOTE: This is subject to change, the number of items will need to be collected across all buckets if we add
		// support for testing backups/restores with multiple buckets
		ItemsNum: decoded.Backups[0].Buckets[0].Items,
	}

	return backupInfo, nil
}

// restoreBackup will run a restore of the backups in the repository, realistically there should only be a single
// backup.
func (b *BackupClient) restoreBackup(config *value.BenchmarkConfig, cluster *Cluster) error {
	fields := log.Fields{
		"blackhole": config.CBMConfig.Blackhole,
		"hosts":     cluster.hosts(),
	}

	log.WithFields(fields).Info("Restoring backup")

	command := config.CBMConfig.CommandRestore(cluster.ConnectionString(config.CBMConfig.TLS))

	_, err := b.node.client.ExecuteCommand(command)

	return err
}

// purgeArchive ensures our workspace is clean, we don't want any existing files to get in the way.
func (b *BackupClient) purgeArchive(config *value.BenchmarkConfig) error {
	if !strings.HasPrefix(config.CBMConfig.Archive, "s3://") {
		log.WithField("archive", config.CBMConfig.Archive).Info("Purging local archive")
		return b.node.client.RemoveDirectory(config.CBMConfig.Archive)
	}

	log.WithField("archive", config.CBMConfig.Archive).Info("Purging remote archive")

	var command string

	if config.CBMConfig.ObjAccessKeyID != "" {
		command += fmt.Sprintf("export AWS_ACCESS_KEY_ID=%s; ", config.CBMConfig.ObjAccessKeyID)
	}

	if config.CBMConfig.ObjSecretAccessKey != "" {
		command += fmt.Sprintf("export AWS_SECRET_ACCESS_KEY=%s; ", config.CBMConfig.ObjSecretAccessKey)
	}

	if config.CBMConfig.ObjRegion != "" {
		command += fmt.Sprintf("export AWS_REGION=%s; ", config.CBMConfig.ObjRegion)
	}

	command += fmt.Sprintf("aws s3 rm %s --recursive", config.CBMConfig.Archive)

	if config.CBMConfig.ObjEndpoint != "" {
		command += fmt.Sprintf(" --endpoint=%s", config.CBMConfig.ObjEndpoint)
	}

	// We're using S3 backup, use the AWS cli to ensure the remote archive has been removed
	_, err := b.node.client.ExecuteCommand(value.NewCommand(command))
	if err != nil {
		return errors.Wrap(err, "failed to purge remote archive")
	}

	log.WithField("staging_directory", config.CBMConfig.ObjStagingDirectory).Info("Purging local staging directory")

	return b.node.client.RemoveDirectory(config.CBMConfig.ObjStagingDirectory)
}

// purgeBackups uses the remove sub-command to purged all the backups we've created. Note that we use remove instead of
// doing this manually so that we don't have to handle removing cloud data i.e. that's handled by cbbackupmgr.
//
// NOTE: We only want to purge the backups we created and not the whole archive. We might be collecting the logs upon
// completion, therefore, we want all the benchmarks run against the same archive.
func (b *BackupClient) purgeBackups(config *value.BenchmarkConfig) error {
	log.Info("Purging created backups")

	output, err := b.node.client.ExecuteCommand(config.CBMConfig.CommandInfo())
	if err != nil {
		return errors.Wrap(err, "failed to run info")
	}

	type backup struct {
		Date string `json:"date"`
	}

	type overlay struct {
		Backups []backup `json:"backups"`
	}

	var decoded overlay

	err = json.Unmarshal(output, &decoded)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal info output")
	}

	if len(decoded.Backups) == 0 {
		return nil
	}

	_, err = b.node.client.ExecuteCommand(
		config.CBMConfig.CommandRemove(decoded.Backups[0].Date, decoded.Backups[len(decoded.Backups)-1].Date),
	)

	return err
}

// Close the connection to the backup client.
func (b *BackupClient) Close() error {
	return b.node.Close()
}
