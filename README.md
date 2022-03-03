cbtools-autobench
-----------------

[![Go Reference](https://pkg.go.dev/badge/github.com/jamesl33/cbtools-autobench.svg)](https://pkg.go.dev/github.com/jamesl33/cbtools-autobench)

An automatic benchmarking tools designed to benchmark Couchbase tools, written with the intention of producing reliable
benchmarks and to reduce the feedback loop for changes made to performance critical components.

Building
--------

Go modules are used to build `cbtools-autobench`, therefore, building is a simple as running `go build`. For convenience
a `Makefile` has also been provided which can be used via `make build`.

Usage
-----

This tool is broken down into three main components which take place when performing benchmarking:
- Provisioning the cluster
    - Installing dependencies on cluster nodes
    - Installing Couchbase Server
    - Initializing each node
    - Initializing the cluster
    - Creating the test bucket
- Loading benchmarking data
    - Modifying eviction pager (too speed up load times)
    - Loading test data (using multiple instances of `cbbackupmgr`)
    - Resetting the eviction pager (to avoid affecting benchmark results)
    - Compacting the test bucket
- Running benchmarks
    - Backup
    - Restore

Provisioning is done via the `cbtools-autobench provision` sub-command which accepts a configuration (see Configuration
for more information) which describes which servers to user for the backup/cluster nodes.

Loading the benchmarking data will be done the first time provision completes, and may be triggered manually (for
example to load a different dataset without provisioning the cluster again) using the `--load-only` flag.

Benchmarks may be run using the `cbtools-autobench benchmark [backup|restore]` sub-command which accepts a configuration
which indicates the number of benchmark iterations to run, along with the required configuration for `cbbackupmgr`.

Below is an example use case for `cbtools-autobench` using the following configuration:

```yaml
ssh:
  username: "root"
  private_key: "..."
blueprint:
  cluster:
    package_path: "/tmp/couchbase-server-enterprise_7.0.0-5160-ubuntu18.04_amd64.deb"
    nodes:
      - host: "192.168.122.154"
      - host: "192.168.122.68"
    bucket:
      type: "ephemeral"
      eviction_policy: "noEviction"
      data:
        items: 500_000
        size: 25
  backup_client:
    host: "192.168.122.215"
    package_path: "/tmp/couchbase-server-enterprise_7.0.0-5160-ubuntu18.04_amd64.deb"
benchmark:
  iterations: 3
  cbbackupmgr_config:
    archive: "archive"
    repository: "repo"
```

This testing was run locally using virtual machines but may just as easily be run using AWS/GCE by changing the
addresses and ensuring the SSH credentials are valid.

```
$ cbtools-autobench provision -c autobench-config.yaml
2021-05-20T17:37:55Z INFO Establishing ssh connection | {"host":"192.168.122.154"}
2021-05-20T17:37:55Z INFO Establishing ssh connection | {"host":"192.168.122.68"}
2021-05-20T17:37:56Z INFO Successfully established ssh connection | {"host":"192.168.122.154","platform":"ubuntu20.04"}
2021-05-20T17:37:57Z INFO Successfully established ssh connection | {"host":"192.168.122.68","platform":"ubuntu20.04"}
2021-05-20T17:37:57Z INFO Establishing ssh connection | {"host":"192.168.122.215"}
2021-05-20T17:37:58Z INFO Successfully established ssh connection | {"host":"192.168.122.215","platform":"ubuntu20.04"}
2021-05-20T17:37:58Z INFO Provision cluster | {"hosts":["192.168.122.154","192.168.122.68"]}
2021-05-20T17:37:58Z INFO Provisioning node | {"host":"192.168.122.154"}
2021-05-20T17:37:58Z INFO Installing dependencies | {"host":"192.168.122.154"}
2021-05-20T17:37:58Z INFO Provisioning backup client | {"host":"192.168.122.215"}
2021-05-20T17:37:58Z INFO Installing dependencies | {"host":"192.168.122.215"}
2021-05-20T17:37:58Z INFO Provisioning node | {"host":"192.168.122.68"}
2021-05-20T17:37:58Z INFO Installing dependencies | {"host":"192.168.122.68"}
2021-05-20T17:38:05Z INFO Uninstalling 'couchbase-server' | {"host":"192.168.122.68"}
2021-05-20T17:38:05Z INFO Uninstalling 'couchbase-server' | {"host":"192.168.122.215"}
2021-05-20T17:38:05Z INFO Uninstalling 'couchbase-server' | {"host":"192.168.122.154"}
2021-05-20T17:38:08Z INFO Purging install directory | {"host":"192.168.122.215"}
2021-05-20T17:38:08Z INFO Uploading package archive | {"host":"192.168.122.215"}
2021-05-20T17:38:09Z INFO Installing 'couchbase-server' | {"host":"192.168.122.215"}
2021-05-20T17:38:09Z INFO Purging install directory | {"host":"192.168.122.68"}
2021-05-20T17:38:09Z INFO Uploading package archive | {"host":"192.168.122.68"}
2021-05-20T17:38:10Z INFO Installing 'couchbase-server' | {"host":"192.168.122.68"}
2021-05-20T17:38:11Z INFO Purging install directory | {"host":"192.168.122.154"}
2021-05-20T17:38:11Z INFO Uploading package archive | {"host":"192.168.122.154"}
2021-05-20T17:38:11Z INFO Installing 'couchbase-server' | {"host":"192.168.122.154"}
2021-05-20T17:38:45Z INFO Cleaning up package archive | {"host":"192.168.122.154"}
2021-05-20T17:38:45Z INFO Cleaning up package archive | {"host":"192.168.122.215"}
2021-05-20T17:38:46Z INFO Cleaning up package archive | {"host":"192.168.122.68"}
2021-05-20T17:39:15Z INFO Initializing node | {"data_path":"","host":"192.168.122.154"}
2021-05-20T17:39:15Z INFO Disabling 'couchbase-server' | {"host":"192.168.122.215"}
2021-05-20T17:39:16Z INFO Initializing node | {"data_path":"","host":"192.168.122.68"}
2021-05-20T17:39:16Z INFO Initializing cluster | {"hosts":["192.168.122.154","192.168.122.68"],"password":"asdasd","username":"Administrator"}
2021-05-20T17:39:17Z INFO Adding node to cluster | {"host":"192.168.122.154"}
2021-05-20T17:39:17Z INFO Adding node to cluster | {"host":"192.168.122.68"}
2021-05-20T17:39:20Z INFO Rebalancing cluster
2021-05-20T17:39:22Z INFO Creating bucket | {"eviction_policy":"noEviction","name":"default","type":"ephemeral"}
2021-05-20T17:39:52Z INFO Loading test data | {"compact":false}
2021-05-20T17:39:52Z INFO Flushing bucket | {"name":"default"}
2021-05-20T17:40:23Z INFO Modifying eviction percentages | {"hosts":["192.168.122.154","192.168.122.68"]}
2021-05-20T17:40:23Z INFO Modifying eviction percentage on node | {"node":"192.168.122.68","percentage":0}
2021-05-20T17:40:23Z INFO Modifying eviction percentage on node | {"node":"192.168.122.154","percentage":0}
2021-05-20T17:40:23Z INFO Running 'cbbackupmgr' to load data into bucket | {"bucket":"default","host":"192.168.122.154","items":250000,"size":25,"threads":0}
2021-05-20T17:40:23Z INFO Running 'cbbackupmgr' to load data into bucket | {"bucket":"default","host":"192.168.122.68","items":250000,"size":25,"threads":0}
2021-05-20T17:40:25Z INFO Modifying eviction percentages | {"hosts":["192.168.122.154","192.168.122.68"]}
2021-05-20T17:40:25Z INFO Modifying eviction percentage on node | {"node":"192.168.122.154","percentage":30}
2021-05-20T17:40:25Z INFO Modifying eviction percentage on node | {"node":"192.168.122.68","percentage":30}

$ cbtools-autobench benchmark backup -c autobench-config.yaml
2021-05-20T18:04:20Z INFO Establishing ssh connection | {"host":"192.168.122.154"}
2021-05-20T18:04:20Z INFO Establishing ssh connection | {"host":"192.168.122.68"}
2021-05-20T18:04:21Z INFO Successfully established ssh connection | {"host":"192.168.122.154","platform":"ubuntu20.04"}
2021-05-20T18:04:21Z INFO Successfully established ssh connection | {"host":"192.168.122.68","platform":"ubuntu20.04"}
2021-05-20T18:04:21Z INFO Establishing ssh connection | {"host":"192.168.122.215"}
2021-05-20T18:04:22Z INFO Successfully established ssh connection | {"host":"192.168.122.215","platform":"ubuntu20.04"}
2021-05-20T18:04:22Z INFO Beginning 'cbbackupmgr' backup benchmark(s) | {"iterations":3}
2021-05-20T18:04:22Z INFO Purging local archive | {"archive":"archive"}
2021-05-20T18:04:22Z INFO Creating repository
2021-05-20T18:04:22Z INFO Beginning 'cbbackupmgr' backup benchmark | {"iteration":1}
2021-05-20T18:04:22Z INFO Running cluster pre-benchmark tasks
2021-05-20T18:04:22Z INFO Flushing caches | {"hosts":["192.168.122.154","192.168.122.68"]}
2021-05-20T18:04:22Z INFO Running backup client pre-benchmark tasks
2021-05-20T18:04:22Z INFO Creating backup | {"blackhole":false,"hosts":["192.168.122.154","192.168.122.68"]}
2021-05-20T18:04:27Z INFO Purging created backups
2021-05-20T18:04:27Z INFO Beginning 'cbbackupmgr' backup benchmark | {"iteration":2}
2021-05-20T18:04:27Z INFO Running cluster pre-benchmark tasks
2021-05-20T18:04:27Z INFO Flushing caches | {"hosts":["192.168.122.154","192.168.122.68"]}
2021-05-20T18:04:27Z INFO Running backup client pre-benchmark tasks
2021-05-20T18:04:27Z INFO Creating backup | {"blackhole":false,"hosts":["192.168.122.154","192.168.122.68"]}
2021-05-20T18:04:32Z INFO Purging created backups
2021-05-20T18:04:32Z INFO Beginning 'cbbackupmgr' backup benchmark | {"iteration":3}
2021-05-20T18:04:32Z INFO Running cluster pre-benchmark tasks
2021-05-20T18:04:32Z INFO Flushing caches | {"hosts":["192.168.122.154","192.168.122.68"]}
2021-05-20T18:04:32Z INFO Running backup client pre-benchmark tasks
2021-05-20T18:04:32Z INFO Creating backup | {"blackhole":false,"hosts":["192.168.122.154","192.168.122.68"]}
2021-05-20T18:04:37Z INFO Purging created backups
2021-05-20T18:04:37Z INFO Getting bucket stats | {"host":"192.168.122.154"}
| Cluster
| -------
| Node | Version    | Host            |
| 1    | 7.0.0-5160 | 192.168.122.154 |
| 2    | 7.0.0-5160 | 192.168.122.68  |

| Bucket
| ------
| vBuckets | Type      | Eviction Policy | Compact |
| default  | ephemeral | noEviction      | false   |

| Data
| ----
| Items   | Size | Compressible | Load Threads |
| 500,000 | 25B  | false        | auto         |

| Stats
| -----
| Item Count | Memory Used | Disk Used | Residency Ratio |
| 500,000    | 87.94MiB    | 70B       | 100%            |

| Backup Client
| -------------
| Version    | Host            |
| 7.0.0-5160 | 192.168.122.215 |

| CBM
| ----
| Archive | Repository  | Staging Directory | Storage | Threads | Blackhole |
| archive | repo        | N/A               | default | auto    | false     |

| Overview
| --------
| Avg Duration | Avg Size (ADS) | Avg Size (GDS) | Avg Transfer Rate (ADS) | Avg Transfer Rate (GDS) |
| 5.019s       | 223.66MiB      | 11.92MiB       | 52.19MiB/s              | 2.78MiB/s               |

| Rundown
| -------
| Iteration | Duration | Size (ADS) | Size (GDS) | Transfer Rate (ADS) | Transfer Rate (GDS) |
| 1         | 5.316s   | 223.66MiB  | 11.92MiB   | 44.73MiB/s          | 2.38MiB/s           |
| 2         | 4.834s   | 223.66MiB  | 11.92MiB   | 55.91MiB/s          | 2.98MiB/s           |
| 3         | 4.907s   | 223.66MiB  | 11.92MiB   | 55.91MiB/s          | 2.98MiB/s           |

$ cbtools-autobench benchmark restore -c autobench-config.yaml
2021-05-20T18:05:51Z INFO Establishing ssh connection | {"host":"192.168.122.154"}
2021-05-20T18:05:51Z INFO Establishing ssh connection | {"host":"192.168.122.68"}
2021-05-20T18:05:52Z INFO Successfully established ssh connection | {"host":"192.168.122.154","platform":"ubuntu20.04"}
2021-05-20T18:05:52Z INFO Successfully established ssh connection | {"host":"192.168.122.68","platform":"ubuntu20.04"}
2021-05-20T18:05:52Z INFO Establishing ssh connection | {"host":"192.168.122.215"}
2021-05-20T18:05:54Z INFO Successfully established ssh connection | {"host":"192.168.122.215","platform":"ubuntu20.04"}
2021-05-20T18:05:54Z INFO Beginning 'cbbackupmgr' restore benchmark(s) | {"iterations":3}
2021-05-20T18:05:54Z INFO Purging local archive | {"archive":"archive"}
2021-05-20T18:05:54Z INFO Creating repository
2021-05-20T18:05:54Z INFO Creating backup | {"blackhole":false,"hosts":["192.168.122.154","192.168.122.68"]}
2021-05-20T18:05:58Z INFO Beginning 'cbbackupmgr' restore benchmark | {"iteration":1}
2021-05-20T18:05:58Z INFO Flushing bucket | {"name":"default"}
2021-05-20T18:06:29Z INFO Running cluster pre-benchmark tasks
2021-05-20T18:06:29Z INFO Flushing caches | {"hosts":["192.168.122.154","192.168.122.68"]}
2021-05-20T18:06:29Z INFO Running backup client pre-benchmark tasks
2021-05-20T18:06:29Z INFO Restoring backup | {"blackhole":false,"hosts":["192.168.122.154","192.168.122.68"]}
2021-05-20T18:06:33Z INFO Beginning 'cbbackupmgr' restore benchmark | {"iteration":2}
2021-05-20T18:06:33Z INFO Flushing bucket | {"name":"default"}
2021-05-20T18:07:04Z INFO Running cluster pre-benchmark tasks
2021-05-20T18:07:04Z INFO Flushing caches | {"hosts":["192.168.122.154","192.168.122.68"]}
2021-05-20T18:07:04Z INFO Running backup client pre-benchmark tasks
2021-05-20T18:07:04Z INFO Restoring backup | {"blackhole":false,"hosts":["192.168.122.154","192.168.122.68"]}
2021-05-20T18:07:08Z INFO Beginning 'cbbackupmgr' restore benchmark | {"iteration":3}
2021-05-20T18:07:08Z INFO Flushing bucket | {"name":"default"}
2021-05-20T18:07:38Z INFO Running cluster pre-benchmark tasks
2021-05-20T18:07:38Z INFO Flushing caches | {"hosts":["192.168.122.154","192.168.122.68"]}
2021-05-20T18:07:38Z INFO Running backup client pre-benchmark tasks
2021-05-20T18:07:38Z INFO Restoring backup | {"blackhole":false,"hosts":["192.168.122.154","192.168.122.68"]}
2021-05-20T18:07:42Z INFO Getting bucket stats | {"host":"192.168.122.154"}
| Cluster
| -------
| Node | Version    | Host            |
| 1    | 7.0.0-5160 | 192.168.122.154 |
| 2    | 7.0.0-5160 | 192.168.122.68  |

| Bucket
| ------
| vBuckets | Type      | Eviction Policy | Compact |
| default  | ephemeral | noEviction      | false   |

| Data
| ----
| Items   | Size | Compressible | Load Threads |
| 500,000 | 25B  | false        | auto         |

| Stats
| -----
| Item Count | Memory Used | Disk Used | Residency Ratio |
| 0          | 16.98MiB    | 70B       | 100%            |

| Backup Client
| -------------
| Version    | Host            |
| 7.0.0-5160 | 192.168.122.215 |

| CBM
| ----
| Archive | Repository  | Staging Directory | Storage | Threads | Blackhole |
| archive | repo        | N/A               | default | auto    | false     |

| Overview
| --------
| Avg Duration | Avg Size (ADS) | Avg Size (GDS) | Avg Transfer Rate (ADS) | Avg Transfer Rate (GDS) |
| 3.822s       | 223.66MiB      | 11.92MiB       | 74.55MiB/s              | 3.97MiB/s               |

| Rundown
| -------
| Iteration | Duration | Size (ADS) | Size (GDS) | Transfer Rate (ADS) | Transfer Rate (GDS) |
| 1         | 3.833s   | 223.66MiB  | 11.92MiB   | 74.55MiB/s          | 3.97MiB/s           |
| 2         | 3.787s   | 223.66MiB  | 11.92MiB   | 74.55MiB/s          | 3.97MiB/s           |
| 3         | 3.846s   | 223.66MiB  | 11.92MiB   | 74.55MiB/s          | 3.97MiB/s           |
```

Configuration
-------------

Currently, all sub-commands require a configuration to be provided; this configuration describes multiple things
required for provisioning/benchmarking. For example:
- SSH credentials (for connecting to the remote servers)
- Cluster blueprint (describes the setup of the cluster including benchmarking dataset)
- Backup client blueprint
- Benchmark configuration (number of iterations along with `cbbackupmgr` configuration i.e. environment/threads/flags)

Below is a complete rundown of all the available values which may be set in the configuration file, any and all unknown
configuration will be ignored by `cbtools-autobench`.

```yaml
ssh:
  # Username used when connecting via SSH to all servers, therefore, must be the same (usually 'root')
  username: ""
  # Some cloud providers require authentication via a private key (path to a file on disk)
  private_key: ""
  # Password for the private key (optional)
  private_key_passphrase: ""
blueprint:
  # Describing the cluster/dataset
  cluster:
    # May be one of two things:
    #   1) A build number (will be downloaded from latestbuilds)
    #   2) A path to a package archive i.e. .deb/.rpm
    #
    # Will be installed on all the cluster nodes
    package_path: ""
    # List of nodes which will be used to create the cluster
    nodes:
    # Hostname of the server, used to connect via SSH (may be an IP address)
    - host: ""
    # The path where KV data will be stored, configured using 'node-init' from 'couchbase-cli'
      data_path: ""
    # Describing the benchmarking bucket
    bucket:
      # Conditionally limit the number of vBuckets (zero value disables limit)
      vbuckets: 0
      # The bucket type i.e. couchbase/ephemeral
      type: ""
      # The eviction policy i.e. valueOnly/fullEviction/noEviction/nruEviction
      eviction_policy: ""
      # Whether to compact the bucket after the data load phase completes
      compact: false
      # Whether the bucket should have Point-In-Time capability
      pitr_enabled: false
      # The granularity of Point-In-Time backups
      pitr_granularity: 0
      # The maximum history age of Point-In-Time backups
      pitr_max_history_age: 0
      # Describes the dataset which will be loaded after provisioning (or via '--load-only')
      data:
        # The number of items to load
        # In the context of a PiTR backup, this is the sum of all items in all PiTR snapshots that are included in this
        # backup
        items: 0
        # The number of active items (items in a PiTR snapshot)
        # It is the number of documents that are in a bucket and are mutated at least once per each granularity period
        # so that the total number of mutations (items) in a PiTR backup adds up to the given item number (specified by
        # 'items' parameter).
        active_items: 0
        # The size of each item being loaded (will be uniform)
        size: 0
        # Whether or not the data should be compressible (default is incompressible data)
        compressible: false
        # Number of threads to use when loading data (default is number of vCPUs)
        load_threads: 0
  # Describing the backup client
  backup_client:
    # Hostname of the server, used to connect via SSH (may be an IP address)
    host: ""
    # May be one of two things:
    #   1) A build number (will be downloaded from latestbuilds)
    #   2) A path to a package archive i.e. .deb/.rpm
    #
    # Will be installed on the backup client (will be disabled after install)
    package_path: ""
# Describing the benchmark(s) that will take place
benchmark:
  # How many times to run the benchmark, more iterations will provide more accurate results
  iterations: 0
  # Describing how to use/run 'cbbackupmgr'
  cbbackupmgr_config:
    # A map of key/value pairs which will be set as environment variables when running 'cbbackupmgr'
    environment_variables: {}
    # The value passed to '--archive'
    archive: ""
    # The value passed to '--repository'
    repository: ""
    # The value passed to '--storage' (default is not to supply the flag i.e. use the default)
    storage: ""
    # The value passed to '--obj-staging-dir'
    obj_staging_directory: ""
    # The value passed to '--obj-access-key-id'
    obj_access_key_id: ""
    # The value passed to '--obj-secret-access-key'
    obj_secret_access_key: ""
    # The value passed to '--obj-region'
    obj_region: ""
    # The value passed to '--obj-endpoint'
    obj_endpoint: ""
    # Pass the '--obj-auth-by-instance-metadata' flag
    obj_auth_by_instance_metadata: false
    # Pass the '--no-verify-ssl' flag
    obj_no_ssl_verify: false
    # The value passed to '--s3-log-level'
    s3_log_level: ""
    # Pass the '--s3-force-path-style' flag
    s3_force_path_style: false
    # Pass the '--encrypted' flag
    encrypted: false
    # The value passed to '--passphrase'
    passphrase: ""
    # The value passed to '--encryption-algo'
    encryption_algo: ""
    # The value passed to '--threads' (defaults to '--auto-select-threads')
    threads: 0
    # Pass the '--point-in-time' flag
    pitr: false
    # Pass the '--sink blackhole' flag
    blackhole: false
```

When running benchmarks, it's important that the information in the configuration is accurate, otherwise the generated
benchmarking report may contain some invalid/stale information.

Contributing
------------

To contribute to this repository please feel free to create pull requests on GitHub using a fork of this repository.
Make sure you have configured the git hooks so that the code is linted and formatted before uploading the patch.

For the git hooks the following dependencies are required:

```
gofmt
gofumpt
goimports
golangci-lint
```

Once you have installed the dependencies set the git hooks path by using the command below:

```
git config core.hooksPath .githooks
```

## Coding style

In this section we will cover notes on the exact coding style to use for this codebase. Most of the style rules are
enforced by the linters, so here we will only cover ones that are not.

### Documenting

- All exported functions should have a matching docstring.
- Any non-trivial unexported function should also have a matching docstring. Note this is left up to the developer and
  reviewer consideration.
- Docstrings must end on a full stop (`.`).
- Comments must be wrapped at 120 characters.
- Notes on interesting/unexpected behavior should have a newline before them and use the `// NOTE:` prefix.

License
-------
Copyright 2021 Couchbase Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
