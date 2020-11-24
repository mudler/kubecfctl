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
	"fmt"
	"os"

	"github.com/kyokomi/emoji"
	"github.com/mudler/kubecfctl/pkg/deployments"
	kubernetes "github.com/mudler/kubecfctl/pkg/kubernetes"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var backupCmd = &cobra.Command{
	Use:     "backup <options> [COMPONENT]",
	Short:   "backups a deployment",
	Aliases: []string{"inst"},
	Long: `This command backups a deployment in your cluster
	
To list the available deployments, run:

	$ kubecfctl list

Then to backup a component, simply run:

	$ kubecfctl backup [COMPONENT]
`,
	PreRun: func(cmd *cobra.Command, args []string) {

		viper.BindPFlag("version", cmd.Flags().Lookup("version"))
		viper.BindPFlag("output", cmd.Flags().Lookup("output"))

	},
	Run: func(cmd *cobra.Command, args []string) {
		version := viper.GetString("version")
		output := viper.GetString("output")

		cluster, err := kubernetes.NewCluster(os.Getenv("KUBECONFIG"))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		emoji.Println(cluster.GetPlatform().Describe())
		inst := kubernetes.NewInstaller()

		d, err := deployments.GlobalCatalog.Deployment(args[0], deployments.DeploymentOptions{
			Version: version,
			Timeout: 1000,
		})
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		err = inst.Backup(d, *cluster, output)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	backupCmd.Flags().String("output", "", "backup output directory")
	backupCmd.Flags().String("version", "", "Component version")

	RootCmd.AddCommand(backupCmd)
}
