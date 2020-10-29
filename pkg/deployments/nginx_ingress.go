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

type NginxIngress struct {
	Version   string
	ChartURL  string
	Namespace string
	domain    string

	LB      bool
	Timeout int
}

func (k *NginxIngress) SetDomain(d string) {
	k.domain = d
}

func (k NginxIngress) GetDomain() string {
	return k.domain
}

func (k NginxIngress) Describe() string {
	return emoji.Sprintf(":cloud: NginxIngress version: %s\nNginxIngress chart: %s\n", k.Version, k.ChartURL)
}

func (k NginxIngress) Delete(c kubernetes.Cluster) error {
	return c.Kubectl.CoreV1().Namespaces().Delete(context.Background(), k.Namespace, metav1.DeleteOptions{})
}

func (k NginxIngress) apply(c kubernetes.Cluster, upgrade bool) error {

	action := "install"
	if upgrade {
		action = "upgrade"
	}

	currentdir, _ := os.Getwd()

	// Setup NginxIngress helm values
	var helmArgs []string

	// IF a LB won't assign IP addresses, we will forcefully assign those
	if !k.LB {
		for i, ip := range c.GetPlatform().ExternalIPs() {
			helmArgs = append(helmArgs, "--set controller.service.externalIPs["+strconv.Itoa(i)+"]="+ip)
		}
	}

	if _, err := helpers.RunProc("helm "+action+" nginx-ingress --create-namespace --wait --namespace "+k.Namespace+" "+k.ChartURL+" "+strings.Join(helmArgs, " "), currentdir); err != nil {
		return errors.New("Failed installing NginxIngress")
	}

	return c.WaitForPodBySelectorRunning(k.Namespace, "", k.Timeout)
}

func (k NginxIngress) Deploy(c kubernetes.Cluster) error {

	_, err := c.Kubectl.CoreV1().Namespaces().Get(
		context.Background(),
		k.Namespace,
		metav1.GetOptions{},
	)
	if err == nil {
		return errors.New("Namespace " + k.Namespace + " present already, run 'kubecfctl nginx-ingress delete " + k.Version + "' first")
	}

	emoji.Println(":cloud: Deploying Nginx Ingress")
	return k.apply(c, false)
}

func (k NginxIngress) Upgrade(c kubernetes.Cluster) error {
	_, err := c.Kubectl.CoreV1().Namespaces().Get(
		context.Background(),
		k.Namespace,
		metav1.GetOptions{},
	)
	if err != nil {
		return errors.New("Namespace " + k.Namespace + " not present")
	}

	emoji.Println(":cloud: Upgrade Nginx Ingress")
	return k.apply(c, true)
}
