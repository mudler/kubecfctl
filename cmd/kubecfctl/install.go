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

var installCmd = &cobra.Command{
	Use:     "install [COMPONENT] [VERSION]",
	Short:   "installs the component to that version",
	Aliases: []string{"inst"},
	Long: `This command installs the specified component in your cluster.

Currently there are available two components, "kubecf" and "ingress".
`,
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("eirini", cmd.Flags().Lookup("eirini"))
		viper.BindPFlag("rollback", cmd.Flags().Lookup("rollback"))
		viper.BindPFlag("ingress", cmd.Flags().Lookup("ingress"))
		viper.BindPFlag("debug", cmd.Flags().Lookup("debug"))

	},
	Run: func(cmd *cobra.Command, args []string) {
		eirini := viper.GetBool("eirini")
		rollback := viper.GetBool("rollback")
		ingress := viper.GetBool("ingress")
		debug := viper.GetBool("debug")

		cluster, err := kubernetes.NewCluster(os.Getenv("KUBECONFIG"))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		emoji.Println(cluster.GetPlatform().Describe())
		inst := kubernetes.NewInstaller()

		opt := deployments.DeploymentOptions{Eirini: eirini, Timeout: 1000, Ingress: ingress, Debug: debug}
		d, err := deployments.GlobalCatalog.Deployment(args[1], args[2], opt)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		err = inst.Install(d, *cluster)
		if err != nil {
			fmt.Println(err)
			if rollback {
				emoji.Println(":x: Deployment failed, deleting deployment")
				err = inst.Delete(d, *cluster)
				if err != nil {
					fmt.Println(err)
				}
			}
			os.Exit(1)
		}
	},
}

func init() {
	installCmd.Flags().Bool("eirini", false, "Enable/Disable Eirini")
	installCmd.Flags().Bool("rollback", false, "Automatically rollback a failed deployment")
	installCmd.Flags().Bool("ingress", false, "Enable ingress")

	RootCmd.AddCommand(installCmd)
}
