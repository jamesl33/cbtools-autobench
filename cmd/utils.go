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

	"github.com/jamesl33/cbtools-autobench/value"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// markFlagRequired - Mark the provided flag as required panicking if it was not found.
func markFlagRequired(command *cobra.Command, flag string) {
	err := command.MarkFlagRequired(flag)
	if err != nil {
		panic(err)
	}
}

// readConfig - Utility function to read and decode the autobench config file at the given path.
func readConfig(path string) (*value.AutobenchConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open config file")
	}
	defer file.Close()

	var config *value.AutobenchConfig

	err = yaml.NewDecoder(file).Decode(&config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode config file")
	}

	return config, nil
}
