package deployments

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/hashicorp/go-multierror"
	"github.com/kyokomi/emoji"
	"github.com/mudler/kubecfctl/pkg/helpers"
	"github.com/mudler/kubecfctl/pkg/kubernetes"
	"github.com/pkg/errors"
)

type Carrier struct {
	Version                            string
	ChartURL                           string
	quarksVersion                      string
	RegistryUsername, RegistryPassword string
	Namespace                          string
	domain                             string
	Debug                              bool

	Timeout int
}

func (k *Carrier) SetDomain(d string) {
	k.domain = d
}

func (k Carrier) GetDomain() string {
	return k.domain
}

func (k Carrier) GetVersion() string {
	return k.Version
}

func (k Carrier) Describe() string {
	return emoji.Sprintf(":cloud:Carrier version: %s\n:clipboard: url: %s", k.Version, k.ChartURL)
}

func (k Carrier) Delete(c kubernetes.Cluster) error {

	quarks, err := GlobalCatalog.GetQuarks(k.quarksVersion)
	if err != nil {
		return err
	}
	err = quarks.Delete(c)
	if err != nil {
		return err
	}

	dir, err := ioutil.TempDir(os.TempDir(), "kubecfctl")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	var result error
	if _, err := helpers.RunProc(fmt.Sprintf("git clone %s ./", k.ChartURL), dir, k.Debug); err != nil {
		result = multierror.Append(result, err)
	}

	if _, err := helpers.RunProc("./gitea/uninstall", dir, k.Debug); err != nil {
		result = multierror.Append(result, err)
	}

	if _, err := helpers.RunProc("./kpack/uninstall", dir, k.Debug); err != nil {
		result = multierror.Append(result, err)
	}

	if _, err := helpers.RunProc("./drone/uninstall", dir, k.Debug); err != nil {
		result = multierror.Append(result, err)
	}

	if _, err := helpers.RunProc("./eirini/uninstall", dir, k.Debug); err != nil {
		result = multierror.Append(result, err)
	}
	return nil
}

func (k Carrier) Deploy(c kubernetes.Cluster) error {
	dir, err := ioutil.TempDir(os.TempDir(), "kubecfctl")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	quarks, err := GlobalCatalog.GetQuarks(k.quarksVersion)
	if err != nil {
		return err
	}
	err = quarks.Deploy(c)
	if err != nil {
		return err
	}

	var result error
	out, err := helpers.RunProc(fmt.Sprintf("git clone %s ./", k.ChartURL), dir, k.Debug)
	if err != nil {
		result = multierror.Append(result, err)
	}
	fmt.Println(out)

	out, err = helpers.RunProc(fmt.Sprintf("./gitea/install %s", c.GetPlatform().ExternalIPs()[0]), dir, k.Debug)
	if err != nil {
		result = multierror.Append(result, err)
	}
	fmt.Println(out)
	out, err = helpers.RunProc(fmt.Sprintf("./kpack/install %s %s", k.RegistryUsername, k.RegistryPassword), dir, k.Debug)
	if err != nil {
		result = multierror.Append(result, err)
	}
	fmt.Println(out)
	out, err = helpers.RunProc(fmt.Sprintf("./drone/install %s", c.GetPlatform().ExternalIPs()[0]), dir, k.Debug)
	if err != nil {
		result = multierror.Append(result, err)
	}
	fmt.Println(out)
	out, err = helpers.RunProc("./eirini/install", dir, k.Debug)
	if err != nil {
		result = multierror.Append(result, err)
	}
	fmt.Println(out)
	out, err = helpers.RunProc(fmt.Sprintf("./drone-gitea/install %s", c.GetPlatform().ExternalIPs()[0]), dir, k.Debug)
	if err != nil {
		result = multierror.Append(result, err)
	}
	fmt.Println(out)
	return err
}

func (k Carrier) Upgrade(c kubernetes.Cluster) error {

	if err := k.Delete(c); err != nil {
		return errors.Wrap(err, "while deploying quarks operator")
	}
	emoji.Println(":ship:Upgrading kubecf")

	return k.Deploy(c)
}
