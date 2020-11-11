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

var deleteCmd = &cobra.Command{
	Use:     "delete [VERSION]",
	Short:   "deletes a deployment",
	Aliases: []string{"inst"},
	Long:    `This command deletes a deployment in your cluster`,
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("eirini", cmd.Flags().Lookup("eirini"))
		viper.BindPFlag("ingress", cmd.Flags().Lookup("ingress"))
		viper.BindPFlag("debug", cmd.Flags().Lookup("debug"))
		viper.BindPFlag("version", cmd.Flags().Lookup("version"))
		viper.BindPFlag("chart", cmd.Flags().Lookup("chart"))
		viper.BindPFlag("quarks-chart", cmd.Flags().Lookup("quarks-chart"))
		viper.BindPFlag("additional-namespace", cmd.Flags().Lookup("additional-namespace"))

	},
	Run: func(cmd *cobra.Command, args []string) {
		eirini := viper.GetBool("eirini")
		ingress := viper.GetBool("ingress")
		debug := viper.GetBool("debug")
		version := viper.GetString("version")
		chartURL := viper.GetString("chart")
		quarksChart := viper.GetString("quarks-chart")
		additionalNamespaces := viper.GetStringSlice("additional-namespace")

		cluster, err := kubernetes.NewCluster(os.Getenv("KUBECONFIG"))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		emoji.Println(cluster.GetPlatform().Describe())
		inst := kubernetes.NewInstaller()

		d, err := deployments.GlobalCatalog.Deployment(args[0], deployments.DeploymentOptions{
			Version:              version,
			Eirini:               eirini,
			Timeout:              1000,
			Ingress:              ingress,
			Debug:                debug,
			ChartURL:             chartURL,
			QuarksURL:            quarksChart,
			AdditionalNamespaces: additionalNamespaces,
		})
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
	deleteCmd.Flags().Bool("eirini", false, "Enable/Disable Eirini")
	deleteCmd.Flags().Bool("ingress", false, "Enable ingress")
	deleteCmd.Flags().String("chart", "", "Chart URL (tgz)")
	deleteCmd.Flags().String("quarks-chart", "", "Quarks Chart URL (tgz)")
	deleteCmd.Flags().String("version", "", "Component version to deploy")
	deleteCmd.Flags().StringSlice("additional-namespace", []string{}, "Additional namespaces to watch for (optional, required only by Quarks) ")

	RootCmd.AddCommand(deleteCmd)
}
