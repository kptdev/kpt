// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmdtutorials

import (
	docs "github.com/GoogleContainerTools/kpt/internal/docs/generated/tutorials"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/spf13/cobra"
)

var tutorials = []*cobra.Command{
	{
		Use:   "tutorials-1-get",
		Short: docs.FetchAPackageShort,
		Long:  docs.FetchAPackageLong,
	},
	{
		Use:   "tutorials-2-update",
		Short: docs.UpdateALocalPackageShort,
		Long:  docs.UpdateALocalPackageLong,
	},
	{
		Use:   "tutorials-3-publish",
		Short: docs.PublishAPackageShort,
		Long:  docs.PublishAPackageLong,
	},
	{
		Use:   "tutorials-4-solutions",
		Short: docs.BuildingSolutionsShort,
		Long:  docs.BuildingSolutionsLong,
	},
	{
		Use:   "faq",
		Short: docs.FaqShort,
		Long:  docs.FaqLong,
	},
	{
		Use:   "future-development",
		Short: docs.FutureDevelopmentShort,
		Long:  docs.FutureDevelopmentLong,
	},
}

func Tutorials(parent string) []*cobra.Command {
	for i := range tutorials {
		cmdutil.FixDocs("kpt", parent, tutorials[i])
	}
	return tutorials
}
