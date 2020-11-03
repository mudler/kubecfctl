package deployments

import (
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/kyokomi/emoji"
	"github.com/mudler/kubecfctl/pkg/helpers"
	"github.com/mudler/kubecfctl/pkg/kubernetes"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Stratos struct {
	Version   string
	ChartURL  string
	Namespace string
	domain    string
	Debug     bool

	LB, Ingress bool
	Timeout     int
}

func (k *Stratos) SetDomain(d string) {
	k.domain = d
}

func (k Stratos) GetDomain() string {
	return k.domain
}

func (k Stratos) Describe() string {
	return emoji.Sprintf(":cloud: Stratos version: %s\n:clipboard:Stratos chart: %s\n", k.Version, k.ChartURL)
}

func (k Stratos) Delete(c kubernetes.Cluster) error {
	return c.Kubectl.CoreV1().Namespaces().Delete(context.Background(), k.Namespace, metav1.DeleteOptions{})
}

func (k Stratos) apply(c kubernetes.Cluster, upgrade bool) error {

	action := "install"
	if upgrade {
		action = "upgrade"
	}

	currentdir, _ := os.Getwd()

	// Setup Stratos helm values
	var helmArgs []string

	// IF a LB won't assign IP addresses, we will forcefully assign those
	if !k.LB {
		for i, ip := range c.GetPlatform().ExternalIPs() {
			helmArgs = append(helmArgs, "--set console.service.externalIPs["+strconv.Itoa(i)+"]="+ip)
		}
		helmArgs = append(helmArgs, "--set console.service.servicePort=8443")
		helmArgs = append(helmArgs, "--set console.service.type=LoadBalancer")
	}
	if k.Ingress {
		helmArgs = append(helmArgs, "--set console.service.ingress.enabled=true")
	}

	if _, err := helpers.RunProc("helm "+action+" stratos --create-namespace --wait --namespace "+k.Namespace+" "+k.ChartURL+" "+strings.Join(helmArgs, " "), currentdir, k.Debug); err != nil {
		return errors.New("Failed installing Stratos")
	}

	return c.WaitForPodBySelectorRunning(k.Namespace, "", k.Timeout)
}

func (k Stratos) Deploy(c kubernetes.Cluster) error {

	_, err := c.Kubectl.CoreV1().Namespaces().Get(
		context.Background(),
		k.Namespace,
		metav1.GetOptions{},
	)
	if err == nil {
		return errors.New("Namespace " + k.Namespace + " present already, run 'kubecfctl nginx-ingress delete " + k.Version + "' first")
	}

	emoji.Println(":cloud: Deploying Stratos")
	return k.apply(c, false)
}

func (k Stratos) Upgrade(c kubernetes.Cluster) error {
	_, err := c.Kubectl.CoreV1().Namespaces().Get(
		context.Background(),
		k.Namespace,
		metav1.GetOptions{},
	)
	if err != nil {
		return errors.New("Namespace " + k.Namespace + " not present")
	}

	emoji.Println(":cloud: Upgrade Stratos")
	return k.apply(c, true)
}
