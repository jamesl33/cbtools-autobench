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
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jamesl33/cbtools-autobench/value"

	"github.com/apex/log"
	"github.com/couchbase/tools-common/sync/hofp"
	"github.com/couchbase/tools-common/utils/maths"
	"github.com/couchbase/tools-common/utils/system"
	"github.com/pkg/errors"
)

// memInfo is a prefix which will be added to commands which require memory/quota based information. For example, when
// provisioning a bucket we will use 80% of the available memory by default.
const memInfo = `
	FREE=$(free | awk '{ print $2 }' | sed '1d;3d' | awk '{ print int($0 / 1024) }');
	QUOTA=$(echo $FREE | awk '{ print int($0 * 0.8) }');
`

// Cluster represents a connection to a number of nodes in a Couchbase Cluster (note that the cluster may not be setup
// yet).
type Cluster struct {
	blueprint *value.ClusterBlueprint
	nodes     []*Node
}

// NewCluster creates a connection to each of the remote cluster nodes using the provided ssh config.
func NewCluster(config *value.SSHConfig, blueprint *value.ClusterBlueprint) (*Cluster, error) {
	var (
		pool  = hofp.NewPool(hofp.Options{Size: maths.Min(system.NumCPU(), len(blueprint.Nodes))})
		nodes = make([]*Node, len(blueprint.Nodes))
	)

	connect := func(idx int, nb *value.NodeBlueprint) error {
		var err error

		nodes[idx], err = NewNode(config, nb)
		if err != nil {
			return err
		}

		return nil
	}

	queue := func(idx int, nb *value.NodeBlueprint) error {
		return pool.Queue(func(_ context.Context) error { return connect(idx, nb) })
	}

	for idx, nb := range blueprint.Nodes {
		if queue(idx, nb) != nil {
			break
		}
	}

	err := pool.Stop()
	if err != nil {
		return nil, errors.Wrap(err, "failed to stop pool")
	}

	return &Cluster{blueprint: blueprint, nodes: nodes}, nil
}

// Provision will provision the cluster installing Couchbase and any required dependencies.
func (c *Cluster) Provision() error {
	log.WithField("hosts", c.hosts()).Info("Provision cluster")

	err := c.provisionNodes()
	if err != nil {
		return errors.Wrap(err, "failed to provision nodes")
	}

	err = c.initializeCB()
	if err != nil {
		return errors.Wrap(err, "failed to initialize Couchbase")
	}

	err = c.enableDeveloperPreviewMode()
	if err != nil {
		return errors.Wrap(err, "failed to enable developer preview mode")
	}

	// Sometimes it's useful to limit the number of vBuckets in the remote cluster when performing testing which is
	// scaled to simulate a dataset of a certain size.
	err = c.limitVBuckets()
	if err != nil {
		return errors.Wrap(err, "failed to limit vBuckets")
	}

	err = c.createBucket()
	if err != nil {
		return errors.Wrap(err, "failed to create bucket")
	}

	// If we request to flush the bucket to close to the creation, we may hit a 500 internal error
	time.Sleep(30 * time.Second)

	return nil
}

// LoadData will load the benchmark dataset using the data loader specified in the config. The load phase is sped up by
// modifying the eviction pager settings to speed up eviction.
func (c *Cluster) LoadData(compact bool) error {
	log.WithField("compact", compact).Info("Loading test data")

	err := c.flushBucket()
	if err != nil {
		return errors.Wrap(err, "failed to flush bucket")
	}

	err = c.modifyEvictionPercentages(0)
	if err != nil {
		return errors.Wrap(err, "failed to set eviction percentages to zero")
	}

	err = c.loadData()
	if err != nil {
		return errors.Wrap(err, "failed to load data")
	}

	err = c.modifyEvictionPercentages(30)
	if err != nil {
		return errors.Wrap(err, "failed to reset eviction percentages")
	}

	if !compact {
		return nil
	}

	err = c.compactBucket()
	if err != nil {
		return errors.Wrap(err, "failed to compact bucket")
	}

	return nil
}

// CollectLogs will collect the logs from the remote cluster then copy the logs into the provided directory.
func (c *Cluster) CollectLogs(path string) ([]string, error) {
	log.WithField("path", path).Info("Collecting cluster logs")

	err := c.startCollection()
	if err != nil {
		return nil, errors.Wrap(err, "failed to start collection")
	}

	// We are safe to ignore the error here since 'logCollectionComplete' does not return an error
	timeout, _ := poll(c.logCollectionComplete, 5*time.Minute)
	if timeout {
		return nil, errors.New("timeout whilst waiting for log collection to complete")
	}

	paths, err := c.collectionPaths()
	if err != nil {
		return nil, errors.Wrap(err, "failed to determine the paths to logs")
	}

	err = c.downloadLogs(paths, path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to download logs")
	}

	converted := make([]string, 0, len(paths))
	for _, logPath := range paths {
		converted = append(converted, filepath.Join(path, filepath.Base(logPath)))
	}

	return converted, nil
}

// Stats returns the basic stats from the cluster as reported by ns_server.
func (c *Cluster) Stats() (*value.Stats, error) {
	log.WithField("host", c.blueprint.Nodes[0].Host).Info("Getting bucket stats")

	// This should probably be done with 'cbrest' or by using an actual HTTP client but for now using curl will suffice
	output, err := exec.Command("curl", "-s", "-u", "Administrator:asdasd",
		fmt.Sprintf("%s:8091/pools/default/buckets/default", c.blueprint.Nodes[0].Host)).CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute curl command")
	}

	type overlay struct {
		BasicStats *value.Stats `json:"basicStats"`
	}

	var decoded overlay

	err = json.Unmarshal(output, &decoded)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal stats")
	}

	return decoded.BasicStats, nil
}

// startCollection uses the CLI to begin a log collection on all the nodes in the cluster.
func (c *Cluster) startCollection() error {
	log.Info("Starting log collection")

	_, err := c.nodes[0].client.ExecuteCommand(
		value.NewCommand(`couchbase-cli collect-logs-start -c %s -u Administrator -p asdasd --all-nodes`,
			c.nodes[0].blueprint.Host))

	return err
}

// compactionComplete returns a boolean indicating whether any compaction tasks are still running on the cluster.
func (c *Cluster) compactionComplete() (bool, error) {
	log.Info("Checking compaction status")

	// This should probably be done with 'cbrest' or by using an actual HTTP client but for now using curl will suffice
	output, err := exec.Command("curl", "-s", "-u", "Administrator:asdasd",
		fmt.Sprintf("%s:8091/pools/default/tasks", c.blueprint.Nodes[0].Host)).CombinedOutput()
	if err != nil {
		return false, errors.Wrap(err, "")
	}

	type overlay struct {
		Type   string `json:"type"`
		Status string `json:"status"`
	}

	var decoded []overlay

	err = json.Unmarshal(output, &decoded)
	if err != nil {
		return false, errors.Wrap(err, "failed to unmarshal response")
	}

	for _, task := range decoded {
		if task.Type == "bucket_compaction" && task.Status == "running" {
			return false, nil
		}
	}

	return len(decoded) == 1 && decoded[0].Type == "rebalance", nil
}

// logCollectionComplete returns a boolean indicating whether the current log collection has completed.
func (c *Cluster) logCollectionComplete() (bool, error) {
	log.Info("Checking log collection status")

	_, err := c.nodes[0].client.ExecuteCommand(value.NewCommand(`couchbase-cli collect-logs-status -c %s \
		-u Administrator -p asdasd | grep -q '^Status: completed'`, c.nodes[0].blueprint.Host))

	return err == nil, nil
}

// collectionPaths returns a slice of the paths to the cbcollect archives.
func (c *Cluster) collectionPaths() ([]string, error) {
	log.Info("Determining which logs to download from cluster")

	output, err := c.nodes[0].client.ExecuteCommand(value.NewCommand(
		`couchbase-cli collect-logs-status -c %s -u Administrator -p asdasd | grep 'path :' | \
			awk '{ print $3 }' | paste -sd ","`, c.nodes[0].blueprint.Host,
	))

	return strings.Split(strings.TrimSpace(string(output)), ","), err
}

func (c *Cluster) downloadLogs(logPaths []string, output string) error {
	log.Info("Downloading cluster logs")

	for _, source := range logPaths {
		err := c.forEachNode(func(node *Node) error {
			if !node.client.FileExists(source) {
				return nil
			}

			sink := filepath.Join(output, filepath.Base(source))

			fields := log.Fields{"host": node.blueprint.Host, "source": source, "sink": sink}
			log.WithFields(fields).Info("Downloading cluster logs from node")

			return node.client.SecureDownload(source, sink)
		})
		if err != nil {
			return errors.Wrapf(err, "failed to download logs at '%s'", source)
		}
	}

	return nil
}

// provisionNodes provisions and initializes Couchbase Server on all the node in the cluster.
func (c *Cluster) provisionNodes() error {
	return c.forEachNode(func(node *Node) error { return c.provisionNode(node) })
}

// provisionNode provision and initialize Couchbase Server on the provided node.
func (c *Cluster) provisionNode(node *Node) error {
	log.WithField("host", node.blueprint.Host).Info("Provisioning node")

	err := node.provision(c.blueprint.PackagePath)
	if err != nil {
		return errors.Wrap(err, "failed to provision node")
	}

	err = node.createDataPath()
	if err != nil {
		return errors.Wrap(err, "failed to create data path")
	}

	err = node.initializeCB()
	if err != nil {
		return errors.Wrap(err, "failed to initialize Couchbase Server")
	}

	return nil
}

// initializeCB will initialize Couchbase Server
func (c *Cluster) initializeCB() error {
	err := c.clusterInit()
	if err != nil {
		return errors.Wrap(err, "failed to initialize cluster")
	}

	err = c.forEachNode(func(node *Node) error { return c.serverAdd(node) })
	if err != nil {
		return errors.Wrap(err, "failed to add cluster nodes")
	}

	err = c.rebalance()
	if err != nil {
		return errors.Wrap(err, "failed to rebalance nodes into cluster")
	}

	return nil
}

// limitVBuckets uses /diag/eval to limit the number of vBuckets in the cluster.
func (c *Cluster) limitVBuckets() error {
	// We're using a default number of vBuckets don't bother changing anything
	if c.blueprint.Bucket.VBuckets == 0 || c.blueprint.Bucket.VBuckets == 1024 {
		return nil
	}

	log.WithField("vbuckets", c.blueprint.Bucket.VBuckets).Info("Limiting number of vBuckets")

	_, err := c.nodes[0].client.ExecuteCommand(value.NewCommand(
		`curl -X POST -u Administrator:asdasd localhost:8091/diag/eval -d \
			"ns_config:set(couchbase_num_vbuckets_default, %d)."`, c.blueprint.Bucket.VBuckets))

	return err
}

// enableDeveloperPreviewMode enables the developer preview mode for the cluster.
func (c *Cluster) enableDeveloperPreviewMode() error {
	if !c.blueprint.DeveloperPreview {
		return nil
	}

	log.WithField("hosts", c.hosts()).Info("Enabling developer preview mode")

	// Using POST request instead of the related CLI command since it prompts for user input confirmation
	_, err := c.nodes[0].client.ExecuteCommand(value.NewCommand(`curl -X POST -u Administrator:asdasd \
		localhost:8091/settings/developerPreview -d "enabled=true"`))

	return err
}

// createBucket creates the benchmarking on the remote cluster which by default uses a quota of 80% of the total memory
// on the cluster nodes.
func (c *Cluster) createBucket() error {
	fields := log.Fields{
		"name":                 "default",
		"type":                 c.blueprint.Bucket.Type,
		"eviction_policy":      c.blueprint.Bucket.EvictionPolicy,
		"pitr_enabled":         c.blueprint.Bucket.PiTREnabled,
		"pitr_granularity":     c.blueprint.Bucket.PiTRGranularity,
		"pitr_max_history_age": c.blueprint.Bucket.PiTRMaxHistoryAge,
	}

	log.WithFields(fields).Info("Creating bucket")

	command := fmt.Sprintf(
		`%s couchbase-cli bucket-create --bucket default --bucket-type %s -c localhost:8091 \
			-u Administrator -p asdasd --bucket-ramsize $QUOTA --bucket-eviction-policy %s \
			--bucket-replica 0 --enable-flush 1 --wait`,
		memInfo,
		c.blueprint.Bucket.Type,
		c.blueprint.Bucket.EvictionPolicy,
	)

	command = c.addPiTRArgs(command)

	_, err := c.nodes[0].client.ExecuteCommand(value.NewCommand(command))

	return err
}

// flushBucket flushes the benchmarking bucket on the remote cluster.
//
// TODO (jamesl33) This looks to be a synchronous operation so for large buckets this operation may timeout and fail.
func (c *Cluster) flushBucket() error {
	log.WithField("name", "default").Info("Flushing bucket")

	_, err := c.nodes[0].client.ExecuteCommand(value.NewCommand(`couchbase-cli bucket-flush -c localhost:8091 \
		-u Administrator -p asdasd --bucket default --force`))
	if err != nil {
		return err
	}

	// We've got to wait for things to complete, this isn't ideal but will have to do for now
	time.Sleep(30 * time.Second)

	return nil
}

// compactBucket compacts the benchmarking bucket on the remote cluster.
func (c *Cluster) compactBucket() error {
	log.WithField("name", "default").Info("Compacting bucket")

	_, err := c.nodes[0].client.ExecuteCommand(value.NewCommand(`couchbase-cli bucket-compact -c localhost:8091 \
		-u Administrator -p asdasd --bucket default`))
	if err != nil {
		return errors.Wrap(err, "")
	}

	// We've got to wait for things to start, for example we need to wait for the compaction entry to be added to the
	// running tasks.
	time.Sleep(30 * time.Second)

	timeout, err := poll(c.compactionComplete, 24*time.Hour)
	if err != nil {
		return errors.Wrap(err, "failed to poll until compaction completed")
	}

	if timeout {
		return errors.New("timeout whilst waiting for bucket compaction to complete")
	}

	return nil
}

// runPreBenchmarkTasks performs any tasks which should be completed prior to running any benchmarks. For example, we
// should flush the caches to avoid skewed results.
func (c *Cluster) runPreBenchmarkTasks() error {
	log.Info("Running cluster pre-benchmark tasks")

	err := c.flushCaches()
	if err != nil {
		return errors.Wrap(err, "failed to flush caches")
	}

	return nil
}

// flushCaches flushes the caches on all the nodes in the cluster.
func (c *Cluster) flushCaches() error {
	log.WithField("hosts", c.hosts()).Info("Flushing caches")

	return c.forEachNode(func(node *Node) error { return node.client.FlushCaches() })
}

// forEachNode is a utility function which concurrently runs the provided function on each node in the cluster.
func (c *Cluster) forEachNode(fn func(node *Node) error) error {
	pool := hofp.NewPool(hofp.Options{
		Size: maths.Min(system.NumCPU(), len(c.nodes)),
	})

	queue := func(node *Node) error { return pool.Queue(func(_ context.Context) error { return fn(node) }) }

	for _, node := range c.nodes {
		if queue(node) != nil {
			break
		}
	}

	return pool.Stop()
}

// modifyEvictionPercentages updates the eviction percentages on each node in the cluster to the given value.
func (c *Cluster) modifyEvictionPercentages(percentage int) error {
	log.WithField("hosts", c.hosts()).Info("Modifying eviction percentages")

	return c.forEachNode(func(node *Node) error { return c.modifyEvictionPercentage(node, percentage) })
}

// modifyEvictionPercentage updates the eviction percentage on the given node to the given value.
func (c *Cluster) modifyEvictionPercentage(node *Node, percentage int) error {
	fields := log.Fields{"node": node.blueprint.Host, "percentage": percentage}
	log.WithFields(fields).Info("Modifying eviction percentage on node")

	_, err := c.nodes[0].client.ExecuteCommand(
		value.NewCommand(`cbepctl localhost:11210 -b default -u Administrator -p asdasd \
			set flush_param item_eviction_age_percentage %d`, percentage))

	return err
}

// loadData runs the data loader specified in the config on each node in the cluster to generate the benchmarking
// dataset.
func (c *Cluster) loadData() error {
	items := make(chan int, len(c.nodes))

	for i := 0; i < len(c.nodes)-1; i++ {
		items <- c.blueprint.Bucket.Data.Items / len(c.nodes)
	}

	items <- (c.blueprint.Bucket.Data.Items / len(c.nodes)) + (c.blueprint.Bucket.Data.Items % len(c.nodes))

	var nodeDataLoadingFunc func(node *Node) error

	switch c.blueprint.Bucket.Data.DataLoader {
	case value.CBM:
		nodeDataLoadingFunc = func(node *Node) error { return c.loadDataFromNodeUsingBackupMgr(node, <-items) }
	case value.Pillowfight:
		nodeDataLoadingFunc = func(node *Node) error { return c.loadDataFromNodeUsingPillowfight(node, <-items) }
	default:
		return fmt.Errorf("unknown/unsupported data loader '%s'", c.blueprint.Bucket.Data.DataLoader)
	}

	return c.forEachNode(nodeDataLoadingFunc)
}

// loadDataFromNodeUsingBackupMgr runs 'cbbackupmgr' on the provided node to load the given number of items into the
// benchmarking bucket.
func (c *Cluster) loadDataFromNodeUsingBackupMgr(node *Node, items int) error {
	fields := log.Fields{
		"host":    node.blueprint.Host,
		"bucket":  "default",
		"items":   items,
		"size":    c.blueprint.Bucket.Data.Size,
		"threads": c.blueprint.Bucket.Data.LoadThreads,
	}

	log.WithFields(fields).Info("Running 'cbbackupmgr' to load data into bucket")

	command := fmt.Sprintf(`cbbackupmgr generate --cluster localhost:8091 -u Administrator --password asdasd \
		--bucket default --num-documents %d --prefix $(cat /dev/urandom | tr -dc 'a-z0-9' | fold -w 5 | head -n 1):: \
		--size %d --no-progress-bar`,
		items,
		c.blueprint.Bucket.Data.Size,
	)

	if c.blueprint.Bucket.Data.LoadThreads != 0 {
		command += fmt.Sprintf(" --threads %d", c.blueprint.Bucket.Data.LoadThreads)
	} else {
		command += " --threads $(nproc)"
	}

	if !c.blueprint.Bucket.Data.Compressible {
		command += " --low-compression"
	}

	_, err := node.client.ExecuteCommand(value.NewCommand(command))

	return err
}

// loadDataFromNodeBackupUsingPillowfight runs 'cbc-pillowfight' on a given node to load and mutate the given number
// of items for at least one time for each granularity period (used with Point-In-Time backup testing).
func (c *Cluster) loadDataFromNodeUsingPillowfight(node *Node, items int) error {
	granularityPeriodsNum := items / c.blueprint.Bucket.Data.ActiveItems
	// Pillowfight can be configured to run a certain number of operations per second but in our case we want it to
	// run a certain number of operations per granularity period (which is at least a second). We work around this
	// limitations by making Pillowfight do one mutation per document per second, which ensures that we have at least
	// one mutation per document for every granularity period that is equal or greater than 1 second.
	//
	// Potential improvement/workaround is discussed in MB-51242.
	cyclesNum := granularityPeriodsNum * int(c.blueprint.Bucket.PiTRGranularity)

	fields := log.Fields{
		"host":         node.blueprint.Host,
		"bucket":       "default",
		"items":        items,
		"active_items": c.blueprint.Bucket.Data.ActiveItems,
		"cycles":       cyclesNum,
		"size":         c.blueprint.Bucket.Data.Size,
		"threads":      c.blueprint.Bucket.Data.LoadThreads,
	}

	log.WithFields(fields).Info("Running 'pillowfight' to load data into bucket")

	command := fmt.Sprintf(`cbc-pillowfight -U localhost -u Administrator -P asdasd -B %d -I %d --num-cycles %d \
		--rate-limit %d -m %d -M %d -r 100 -R --sequential`,
		c.blueprint.Bucket.Data.ActiveItems,
		c.blueprint.Bucket.Data.ActiveItems,
		cyclesNum,
		c.blueprint.Bucket.Data.ActiveItems,
		c.blueprint.Bucket.Data.Size,
		c.blueprint.Bucket.Data.Size,
	)

	if c.blueprint.Bucket.Data.LoadThreads != 0 {
		command += fmt.Sprintf(" --num-threads %d", c.blueprint.Bucket.Data.LoadThreads)
	}

	if !c.blueprint.Bucket.Data.Compressible {
		command += " --compress"
	}

	_, err := node.client.ExecuteCommand(value.NewCommand(command))

	return err
}

// clusterInit uses the CLI to initialize the cluster with an 80% ram quota and the standard cluster_run credentials.
func (c *Cluster) clusterInit() error {
	fields := log.Fields{"hosts": c.hosts(), "username": "Administrator", "password": "asdasd"}
	log.WithFields(fields).Info("Initializing cluster")

	_, err := c.nodes[0].client.ExecuteCommand(value.NewCommand(`
		%s couchbase-cli cluster-init -c localhost:8091 --cluster-username Administrator --cluster-password asdasd \
			--cluster-ramsize $QUOTA`, memInfo))

	return err
}

// serverAdd uses the CLI to add the given node into the cluster.
func (c *Cluster) serverAdd(node *Node) error {
	log.WithField("host", node.blueprint.Host).Info("Adding node to cluster")

	// The first node is already in the cluster, there's nothing to do here
	if c.nodes[0] == node {
		return nil
	}

	_, err := c.nodes[0].client.ExecuteCommand(value.NewCommand(`
		couchbase-cli server-add -c localhost:8091 -u Administrator -p asdasd --server-add %s \
			--server-add-username Administrator --server-add-password asdasd --services data`, node.blueprint.Host))

	return err
}

// rebalance uses the CLI to rebalance the cluster.
func (c *Cluster) rebalance() error {
	log.Info("Rebalancing cluster")

	_, err := c.nodes[0].client.ExecuteCommand(
		value.NewCommand(`couchbase-cli rebalance -c localhost:8091 -u Administrator -p asdasd`))

	return err
}

// addPiTRArgs will conditionally add the PiTR flags to the given command.
func (c *Cluster) addPiTRArgs(command string) string {
	if c.blueprint.Bucket.PiTREnabled {
		command += " --enable-point-in-time 1"
	}

	if c.blueprint.Bucket.PiTRGranularity != 0 {
		command += fmt.Sprintf(" --point-in-time-granularity %d", c.blueprint.Bucket.PiTRGranularity)
	}

	if c.blueprint.Bucket.PiTRMaxHistoryAge != 0 {
		command += fmt.Sprintf(" --point-in-time-max-history-age %d", c.blueprint.Bucket.PiTRMaxHistoryAge)
	}

	return command
}

// ConnectionString returns a connection string which can be used to connect to the cluster.
//
// NOTE: We don't use a multi-node connection string currently since they're not supported until 7.0.0.
func (c *Cluster) ConnectionString() string {
	return fmt.Sprintf("couchbase://%s", c.nodes[0].blueprint.Host)
}

// hosts returns a slice of all the hostnames for the nodes in the cluster.
func (c *Cluster) hosts() []string {
	hosts := make([]string, 0, len(c.nodes))
	for _, node := range c.nodes {
		hosts = append(hosts, node.blueprint.Host)
	}

	return hosts
}

// Close releases any resources in use by the connection.
func (c *Cluster) Close() error {
	return c.forEachNode(func(node *Node) error { return node.Close() })
}

// poll runs the given function until it returns true or we reach the provided timeout.
func poll(pollFunc func() (bool, error), timeout time.Duration) (bool, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	defer cancelFunc()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return true, nil
		case <-ticker.C:
			ready, err := pollFunc()
			if err != nil {
				return false, err
			}

			if ready {
				return false, nil
			}
		}
	}
}
