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

package get

import (
	"fmt"
	"os"

	deployments "github.com/mudler/kubecfctl/pkg/deployments"

	kubernetes "github.com/mudler/kubecfctl/pkg/kubernetes"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var PasswordCmd = &cobra.Command{
	Use:     "kubecf-password [version]",
	Short:   "kubecf-password admin password",
	Aliases: []string{"pw"},
	Long:    `Retrieve CF admin password from KubeCF deployment`,

	RunE: func(cmd *cobra.Command, args []string) error {
		cluster, err := kubernetes.NewCluster(os.Getenv("KUBECONFIG"))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		kubecf, err := deployments.GetKubeCF(args[0])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		pwd, err := kubecf.GetPassword(*cluster)
		if err != nil {
			return errors.Wrap(err, "couldn't find password secret")
		}
		fmt.Printf(string(pwd))

		return nil
	},
}
