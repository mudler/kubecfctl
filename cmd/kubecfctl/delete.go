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
)

var deleteCmd = &cobra.Command{
	Use:     "delete [VERSION]",
	Short:   "deletes kubecf",
	Aliases: []string{"inst"},
	Long:    `This command deletes kubecf in your cluster`,
	Run: func(cmd *cobra.Command, args []string) {
		cluster, err := kubernetes.NewCluster(os.Getenv("KUBECONFIG"))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		emoji.Println(cluster.GetPlatform().Describe())

		inst := kubernetes.NewInstaller()

		opt := deployments.DeploymentOptions{}
		d, err := deployments.GlobalCatalog.Deployment(args[1], args[2], opt)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		err = inst.Delete(d, *cluster)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {

	RootCmd.AddCommand(deleteCmd)
}
