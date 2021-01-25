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
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/apex/log"
)

// signalHandler - Spawn a goroutine which gracefully handles SIGINT by cancelling the returned context, this can be
// used to determine if we need to gracefully terminate.
func signalHandler() context.Context {
	ctx, cancelFunc := context.WithCancel(context.Background())

	signalStream := make(chan os.Signal, 1)
	signal.Notify(signalStream, syscall.SIGINT)

	go func() {
		<-signalStream

		signal.Stop(signalStream)
		close(signalStream)

		log.Warn("Received interrupt signal, gracefully terminating")

		cancelFunc()
	}()

	return ctx
}
