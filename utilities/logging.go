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

package utilities

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/pkg/errors"
)

// levels maps log levels to their string representation.
var levels = map[int]string{
	int(log.DebugLevel): "DEBU",
	int(log.InfoLevel):  "INFO",
	int(log.WarnLevel):  "WARN",
	int(log.ErrorLevel): "ERRO",
	int(log.FatalLevel): "FATA",
}

// LoggingHandler which implements the apex logging handler interface.
type LoggingHandler struct {
	mu     sync.Mutex
	writer io.Writer
}

// NewLoggingHandler creates a new LoggingHandler which will log to stdout.
func NewLoggingHandler() *LoggingHandler {
	return &LoggingHandler{
		writer: os.Stdout,
	}
}

// HandleLog implements the handler interface for the apex logging module.
func (h *LoggingHandler) HandleLog(e *log.Entry) error {
	fields, err := json.Marshal(e.Fields)
	if err != nil {
		return errors.Wrap(err, "failed to marshal fields")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	timestamp := time.Now().UTC().Format(time.RFC3339)

	if len(fields) == 0 || string(fields) == "{}" {
		fmt.Fprintf(h.writer, "%s %s %s\n", timestamp, levels[int(e.Level)], e.Message)
	} else {
		fmt.Fprintf(h.writer, "%s %s %s | %s\n", timestamp, levels[int(e.Level)], e.Message, fields)
	}

	return nil
}
