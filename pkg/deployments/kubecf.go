package deployments

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/kyokomi/emoji"
	"github.com/mudler/kubecfctl/pkg/helpers"
	"github.com/mudler/kubecfctl/pkg/kubernetes"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type KubeCF struct {
	Version        string
	ChartURL       string
	QuarksOperator string
	Namespace      string
	domain         string

	Eirini, Ingress, Autoscaler bool
	Timeout                     int
}

var AvailableKubeCFVersions = map[string]KubeCF{
	"2.5.8": KubeCF{
		Version:        "2.5.8",
		ChartURL:       "https://github.com/cloudfoundry-incubator/kubecf/releases/download/v2.5.8/kubecf-v2.5.8.tgz",
		Namespace:      "kubecf",
		QuarksOperator: "https://github.com/cloudfoundry-incubator/quarks-operator/releases/download/v6.1.17/cf-operator-6.1.17+0.gec409fd7.tgz",
	},
}

func GetKubeCF(version string) (*KubeCF, error) {
	kubecf, ok := AvailableKubeCFVersions[version]
	if !ok {
		return nil, errors.New("Unsupported KubeCF version")
	}
	return &kubecf, nil
}

func (k *KubeCF) SetDomain(d string) {
	k.domain = d
}

func (k KubeCF) GetDomain() string {
	return k.domain
}

func (k KubeCF) Delete(c kubernetes.Cluster) error {
	currentdir, _ := os.Getwd()

	helpers.RunProc("kubectl delete crds boshdeployments.quarks.cloudfoundry.org", currentdir)
	helpers.RunProc("kubectl delete crds quarksjobs.quarks.cloudfoundry.org", currentdir)
	helpers.RunProc("kubectl delete crds quarkssecrets.quarks.cloudfoundry.org", currentdir)
	helpers.RunProc("kubectl delete crds quarksstatefulsets.quarks.cloudfoundry.org", currentdir)

	c.Kubectl.CoreV1().Namespaces().Delete(context.Background(), "kubecf", metav1.DeleteOptions{})
	c.Kubectl.CoreV1().Namespaces().Delete(context.Background(), "cf-operator", metav1.DeleteOptions{})
	c.Kubectl.CoreV1().Namespaces().Delete(context.Background(), "eirini", metav1.DeleteOptions{})

	return nil
}

func (k KubeCF) Deploy(c kubernetes.Cluster) error {
	emoji.Println(":anchor: Deploying cf-operator")
	_, err := c.Kubectl.CoreV1().Namespaces().Get(
		context.Background(),
		"cf-operator",
		metav1.GetOptions{},
	)
	if err == nil {
		return errors.New("Namespace 'cf-operator' present already, run 'kubecfctl delete " + k.Version + "' first")
	}
	currentdir, _ := os.Getwd()
	out, err := helpers.RunProc("helm install cf-operator --create-namespace --namespace cf-operator --wait "+k.QuarksOperator+" --set global.singleNamespace.name="+k.Namespace, currentdir)
	fmt.Println(out)
	if err != nil {
		return errors.New("Failed installing cf-operator")
	}

	if err := c.WaitForPodBySelectorRunning("cf-operator", "", 900); err != nil {
		return errors.Wrap(err, "failed waiting")
	}

	// Setup KubeCF helm values
	var helmArgs []string
	helmArgs = append(helmArgs, "--set system_domain="+k.domain)

	if k.Eirini {
		fmt.Println("Deploying kubecf with Eirini enabled")
		helmArgs = append(helmArgs, "--set features.eirini.enabled=true")
		helmArgs = append(helmArgs, "--set install_stacks[0]=sle15")
	}

	if !k.Ingress {
		for _, s := range []string{"router", "tcp-router", "ssh-proxy"} {
			helmArgs = append(helmArgs, "--set services."+s+".type=LoadBalancer")
			for i, ip := range c.GetPlatform().ExternalIPs() {
				helmArgs = append(helmArgs, "--set services."+s+".externalIPs["+strconv.Itoa(i)+"]="+ip)
			}
		}
	} else {
		helmArgs = append(helmArgs, "--set features.ingress.enabled=true")
	}

	if k.Autoscaler {
		helmArgs = append(helmArgs, "--set features.autoscaler.enabled=true")
	}

	// End helm values setup

	emoji.Println("Quarks Operator deployed correctly :check_mark:")

	emoji.Println(":anchor: Deploying kubecf")

	_, err = helpers.RunProc("helm install kubecf --namespace "+k.Namespace+" "+k.ChartURL+" "+strings.Join(helmArgs, " "), currentdir)
	if err != nil {
		return errors.New("Failed installing kubecf")
	}
	emoji.Println(":person_in_bed: Waiting for kubecf to be ready")

	for _, s := range []string{"api", "nats", "cc-worker", "doppler"} {
		//spin := spinner.New(spinner.CharSets[11], 100*time.Millisecond) // Build our new spinner
		//spin.Start()                                                    // Start the spinner
		//helpers.RunProc("kubectl wait --for=condition=Ready pod --namespace kubecf -l quarks.cloudfoundry.org/quarks-statefulset-name="+s+" --timeout="+strconv.Itoa(k.Timeout), currentdir)
		//spin.Stop()
		err = c.WaitUntilPodBySelectorExist(k.Namespace, "quarks.cloudfoundry.org/quarks-statefulset-name="+s, k.Timeout)
		if err != nil {
			return errors.Wrap(err, "Failed waiting for api")
		}
	}

	return c.WaitForPodBySelectorRunning(k.Namespace, "app.kubernetes.io/name=kubecf", k.Timeout)
}
