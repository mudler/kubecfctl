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
	Use:     "install [VERSION]",
	Short:   "installs kubecf",
	Aliases: []string{"inst"},
	Long:    `This command installs kubecf in your cluster`,
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("eirini", cmd.Flags().Lookup("eirini"))
		viper.BindPFlag("rollback", cmd.Flags().Lookup("rollback"))
		viper.BindPFlag("ingress", cmd.Flags().Lookup("ingress"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		eirini := viper.GetBool("eirini")
		rollback := viper.GetBool("rollback")
		ingress := viper.GetBool("ingress")

		cluster, err := kubernetes.NewCluster(os.Getenv("KUBECONFIG"))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		emoji.Println(cluster.GetPlatform().Describe())

		inst := kubernetes.NewInstaller()
		kubecf, err := deployments.GetKubeCF(args[0])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		kubecf.Eirini = eirini
		kubecf.Timeout = 10000
		kubecf.Ingress = ingress
		err = inst.Install(kubecf, *cluster)
		if err != nil {
			fmt.Println(err)
			if rollback {
				emoji.Println(":x: Deployment failed, deleting deployment")
				err = inst.Delete(kubecf, *cluster)
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
