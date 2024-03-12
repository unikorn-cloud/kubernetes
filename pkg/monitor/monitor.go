/*
Copyright 2022-2024 EscherCloud.
Copyright 2024 the Unikorn Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package monitor

import (
	"context"
	"time"

	"github.com/spf13/pflag"

	upgradecluster "github.com/unikorn-cloud/unikorn/pkg/monitor/upgrade/cluster"
	upgradeclustermanager "github.com/unikorn-cloud/unikorn/pkg/monitor/upgrade/clustermanager"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Options allow modification of parameters via the CLI.
type Options struct {
	// pollPeriod defines how often to run.  There's no harm in having it
	// run with high frequency, reads are all cached.  It's mostly down to
	// burning CPU unnecessarily.
	pollPeriod time.Duration
}

// AddFlags registers option flags with pflag.
func (o *Options) AddFlags(flags *pflag.FlagSet) {
	flags.DurationVar(&o.pollPeriod, "poll-period", time.Minute, "Period to poll for updates")
}

// Checker is an interface that monitors must implement.
type Checker interface {
	// Check does whatever the checker is checking for.
	Check(context.Context) error
}

// Run sits in an infinite loop, polling every so often.
func Run(ctx context.Context, c client.Client, o *Options) {
	log := log.FromContext(ctx)

	ticker := time.NewTicker(o.pollPeriod)
	defer ticker.Stop()

	checkers := []Checker{
		upgradecluster.New(c),
		upgradeclustermanager.New(c),
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, checker := range checkers {
				if err := checker.Check(ctx); err != nil {
					log.Error(err, "check failed")
				}
			}
		}
	}
}
