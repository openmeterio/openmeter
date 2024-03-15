// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"runtime/debug"

	"github.com/spf13/cobra"
)

// Provisioned by ldflags.
var version string

//nolint:gochecknoglobals
var (
	revision     string
	revisionDate string
)

//nolint:gochecknoinits,goconst
func init() {
	if version == "" {
		version = "unknown"
	}

	buildInfo, _ := debug.ReadBuildInfo()

	revision = "unknown"
	revisionDate = "unknown"

	for _, setting := range buildInfo.Settings {
		if setting.Key == "vcs.revision" {
			revision = setting.Value
		}

		if setting.Key == "vcs.time" {
			revisionDate = setting.Value
		}
	}
}

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("%s version %s (%s) built on %s\n", "Open Meter", version, revision, revisionDate)
		},
	}
}
