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

var restoreCmd = &cobra.Command{
	Use:     "restore <options> [COMPONENT]",
	Short:   "restores a deployment",
	Aliases: []string{"inst"},
	Long: `This command restores a deployment in your cluster
	
To list the available deployments, run:

	$ kubecfctl list

Then to restore a component, simply run:

	$ kubecfctl restore [COMPONENT]
`,
	PreRun: func(cmd *cobra.Command, args []string) {

		viper.BindPFlag("version", cmd.Flags().Lookup("version"))
		viper.BindPFlag("output", cmd.Flags().Lookup("output"))

		viper.BindPFlag("eirini", cmd.Flags().Lookup("eirini"))
		viper.BindPFlag("rollback", cmd.Flags().Lookup("rollback"))
		viper.BindPFlag("ingress", cmd.Flags().Lookup("ingress"))
		viper.BindPFlag("debug", cmd.Flags().Lookup("debug"))
		viper.BindPFlag("chart", cmd.Flags().Lookup("chart"))
		viper.BindPFlag("quarks-chart", cmd.Flags().Lookup("quarks-chart"))
		viper.BindPFlag("registry-username", cmd.Flags().Lookup("registry-username"))
		viper.BindPFlag("storage-class", cmd.Flags().Lookup("storage-class"))

		viper.BindPFlag("registry-password", cmd.Flags().Lookup("registry-password"))
		viper.BindPFlag("additional-namespace", cmd.Flags().Lookup("additional-namespace"))

	},
	Run: func(cmd *cobra.Command, args []string) {
		eirini := viper.GetBool("eirini")
		ingress := viper.GetBool("ingress")
		debug := viper.GetBool("debug")
		version := viper.GetString("version")
		chartURL := viper.GetString("chart")
		storageClass := viper.GetString("storage-class")
		quarksChart := viper.GetString("quarks-chart")
		registryUserame := viper.GetString("registry-username")
		registryPassword := viper.GetString("registry-password")
		additionalNamespaces := viper.GetStringSlice("additional-namespace")
		output := viper.GetString("output")

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
			RegistryUsername:     registryUserame,
			StorageClass:         storageClass,
			RegistryPassword:     registryPassword,
			AdditionalNamespaces: additionalNamespaces,
		})
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		err = inst.Restore(d, *cluster, output)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	restoreCmd.Flags().String("output", "", "restore output directory")
	restoreCmd.Flags().String("version", "", "Component version")
	restoreCmd.Flags().Bool("eirini", false, "Enable/Disable Eirini")
	restoreCmd.Flags().Bool("rollback", false, "Automatically rollback a failed deployment")
	restoreCmd.Flags().Bool("ingress", false, "Enable ingress")
	restoreCmd.Flags().String("chart", "", "Chart URL (tgz)")
	restoreCmd.Flags().String("quarks-chart", "", "Quarks Chart URL (tgz)")
	restoreCmd.Flags().String("registry-username", "", "Registry username (optional, required only by Carrier)")
	restoreCmd.Flags().String("registry-password", "", "Registry password (optional, required only by Carrier) ")
	restoreCmd.Flags().StringSlice("additional-namespace", []string{}, "Additional namespaces to watch for (optional, required only by Quarks) ")
	restoreCmd.Flags().String("storage-class", "", "Storage class to be used")

	RootCmd.AddCommand(restoreCmd)
}
