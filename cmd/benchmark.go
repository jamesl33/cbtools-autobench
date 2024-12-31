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

package cmd

import (
	"os"

	fsutil "github.com/couchbase/tools-common/fs/util"
	"github.com/jamesl33/cbtools-autobench/nodes"
	"github.com/jamesl33/cbtools-autobench/report"
	"github.com/jamesl33/cbtools-autobench/value"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// benchmarkOptions encapsulates the possible options which can be used to change the behavior of the 'benchmark'
// sub-command.
var benchmarkOptions = struct {
	configPath string
	logsPath   string
	jsonOut    bool
}{}

// benchmarkCommand is the benchmark sub-command, used to benchmark the 'cbbackupmgr' tool by running multiple
// backups/restores against an already provisioned cluster.
var benchmarkCommand = &cobra.Command{
	RunE:      benchmark,
	Short:     "benchmark the cbbackupmgr tool performing either a backup or restore",
	Use:       "benchmark {backup|restore}",
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	ValidArgs: []string{"backup", "restore"},
}

// init the flags/arguments for the benchmark sub-command.
func init() {
	benchmarkCommand.Flags().StringVarP(
		&benchmarkOptions.configPath,
		"config",
		"c",
		"",
		"path to a cbtools-autobench config file",
	)

	benchmarkCommand.Flags().StringVarP(
		&benchmarkOptions.logsPath,
		"collect-logs",
		"l",
		"",
		"collect cluster/cbbackupmgr logs and download them into this directory",
	)

	benchmarkCommand.Flags().BoolVarP(
		&benchmarkOptions.jsonOut,
		"json",
		"j",
		false,
		"JSON format benchmarking report",
	)

	markFlagRequired(benchmarkCommand, "config")
}

// benchmark sub-command, this will use the provided configuration to run one or more benchmarks against an already
// provisioned cluster and then print a report to stdout.
//
// NOTE: The report prints information about the cluster/dataset, therefore, it's up to the user to the dataset hasn't
// changed since it was provisioned.
func benchmark(_ *cobra.Command, args []string) error {
	config, err := readConfig(benchmarkOptions.configPath)
	if err != nil {
		return errors.Wrap(err, "failed to read autobench config")
	}

	cluster, err := nodes.NewCluster(config.SSHConfig, config.Blueprint.Cluster)
	if err != nil {
		return errors.Wrap(err, "failed to connect to cluster")
	}
	defer cluster.Close()

	client, err := nodes.NewBackupClient(config.SSHConfig, config.Blueprint.BackupClient)
	if err != nil {
		return errors.Wrap(err, "failed to connect to backup client")
	}
	defer client.Close()

	ctx := signalHandler()

	var results value.BenchmarkResults

	switch args[0] {
	case "backup":
		results, err = client.BenchmarkBackup(ctx, config.BenchmarkConfig, cluster)
	case "restore":
		results, err = client.BenchmarkRestore(ctx, config.BenchmarkConfig, cluster)
	}

	if err != nil {
		return errors.Wrap(err, "failed to run benchmark(s)")
	}

	stats, err := cluster.Stats()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster stats")
	}

	clusterLogs, backupLogs, err := collectLogs(cluster, client, config.BenchmarkConfig, benchmarkOptions.logsPath)
	if err != nil {
		return errors.Wrap(err, "failed to collect logs")
	}

	report := report.NewReport(report.Options{
		Blueprint:   config.Blueprint,
		Stats:       stats,
		CBMConfig:   config.BenchmarkConfig.CBMConfig,
		Results:     results,
		ClusterLogs: clusterLogs,
		BackupLogs:  backupLogs,
	})

	err = report.Print(benchmarkOptions.jsonOut)
	if err != nil {
		return errors.Wrap(err, "failed to display report")
	}

	return nil
}

// collectLogs will collect the logs from the cluster/backup archive, note if an empty path is provided the logs will
// not be collected.
func collectLogs(cluster *nodes.Cluster, client *nodes.BackupClient, config *value.BenchmarkConfig,
	path string,
) ([]string, string, error) {
	// We haven't been provided a path by the user, this indicates that they don't want to collect the logs
	if path == "" {
		return nil, "", nil
	}

	stats, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, "", errors.Wrap(err, "failed to get info about logs output path")
	}

	if stats != nil && !stats.IsDir() {
		return nil, "", errors.New("logs output path must not exist, or be a directory")
	}

	err = fsutil.Mkdir(path, 0, true, true)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to create logs output directory")
	}

	clusterLogs, err := cluster.CollectLogs(path)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to collect cluster logs")
	}

	backupLogs, err := client.CollectLogs(config, path)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to collect cbbackupmgr logs")
	}

	return clusterLogs, backupLogs, nil
}
