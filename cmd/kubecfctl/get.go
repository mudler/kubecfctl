/*
Copyright Ettore Di Giacinto <mudler@gentoo.org>.
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

package cmd

import (
	get "github.com/mudler/kubecfctl/cmd/kubecfctl/get"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:     "get",
	Short:   "get deployment data",
	Aliases: []string{"g"},
	Long: `This command gets various information from deployments.
	
To get the KubeCF administrator password, run:

	$ kubecfctl get password
`,
}

func init() {
	getCmd.AddCommand(get.PasswordCmd)
	RootCmd.AddCommand(getCmd)
}
