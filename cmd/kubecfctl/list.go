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
	"os"

	"github.com/jedib0t/go-pretty/table"
	"github.com/mudler/kubecfctl/pkg/deployments"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list [COMPONENT]",
	Short:   "lists available deployments",
	Aliases: []string{"inst"},
	Long:    `This command lists available deployments`,
	Run: func(cmd *cobra.Command, args []string) {
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Name", "Version"})

		if len(args) > 0 {
			for _, d := range deployments.GlobalCatalog.Search(args[0]) {
				t.AppendRow(d.([]interface{}))
			}
		} else {
			for _, d := range deployments.GlobalCatalog.GetList() {
				t.AppendRow(d.([]interface{}))
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
