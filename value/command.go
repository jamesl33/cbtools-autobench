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

// Command represents a command to be executed on a system (generally via ssh).
type Command string

// NewCommand creates a new command whilst exposing formatting functionality similar to fmt.Sprintf.
//
// NOTE: Whitespace will be removed from the return command.
func NewCommand(format string, args ...interface{}) Command {
	command := fmt.Sprintf(format, args...)

	command = strings.ReplaceAll(command, "\\\n", "")
	command = strings.ReplaceAll(command, "\n", "")
	command = strings.ReplaceAll(command, "\t", "")

	return Command(command)
}

// ToString converts the provided command into a string which can be directly run on the remote system.
func (c Command) ToString(environment map[string]string) string {
	if len(environment) == 0 {
		return string(c)
	}

	var env string
	for key, value := range environment {
		env += fmt.Sprintf("export %s=%s; ", key, value)
	}

	return env + string(c)
}
