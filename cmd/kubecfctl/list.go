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

	"github.com/jedib0t/go-pretty/table"
	"github.com/kyokomi/emoji"
	"github.com/mudler/kubecfctl/pkg/deployments"
	kubernetes "github.com/mudler/kubecfctl/pkg/kubernetes"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list [COMPONENT]",
	Short:   "lists available deployments",
	Aliases: []string{"inst"},
	Long:    `This command lists available deployments`,
	Run: func(cmd *cobra.Command, args []string) {
		cluster, err := kubernetes.NewCluster(os.Getenv("KUBECONFIG"))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		emoji.Println(cluster.GetPlatform().Describe())

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Name", "Version"})
		if len(args) > 0 {
			switch args[0] {
			case "kubecf":

				for _, d := range deployments.GlobalCatalog.KubeCF {
					t.AppendRow([]interface{}{"KubeCF", d.Version})
				}

			case "nginx-ingress":
				for _, d := range deployments.GlobalCatalog.Nginx {
					t.AppendRow([]interface{}{"nginx-ingresss", d.Version})
				}
			default:
				fmt.Println("Invalid deployment, valid options are: kubecf, nginx-ingress")
			}
		} else {

			for _, d := range deployments.GlobalCatalog.KubeCF {
				t.AppendRow([]interface{}{"kubecf", d.Version})
			}
			for _, d := range deployments.GlobalCatalog.Nginx {
				t.AppendRow([]interface{}{"nginx-ingress", d.Version})
			}
		}

		t.AppendFooter(table.Row{"", "", ""})
		t.SetStyle(table.StyleColoredBright)
		t.Render()
	},
}

func init() {

	RootCmd.AddCommand(listCmd)
}
