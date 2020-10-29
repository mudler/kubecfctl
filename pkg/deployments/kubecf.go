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

	Eirini, Ingress, Autoscaler, LB bool
	Timeout                         int
}

func (k *KubeCF) SetDomain(d string) {
	k.domain = d
}

func (k KubeCF) GetDomain() string {
	return k.domain
}

func (k KubeCF) Describe() string {
	return emoji.Sprintf(":cloud: KubeCF version: %s\nQuarks chart: %s\nKubeCF chart: %s\n", k.Version, k.QuarksOperator, k.ChartURL)
}

func (k KubeCF) Delete(c kubernetes.Cluster) error {
	currentdir, _ := os.Getwd()

	helpers.RunProc("kubectl delete crds boshdeployments.quarks.cloudfoundry.org", currentdir)
	helpers.RunProc("kubectl delete crds quarksjobs.quarks.cloudfoundry.org", currentdir)
	helpers.RunProc("kubectl delete crds quarkssecrets.quarks.cloudfoundry.org", currentdir)
	helpers.RunProc("kubectl delete crds quarksstatefulsets.quarks.cloudfoundry.org", currentdir)

	c.Kubectl.CoreV1().Namespaces().Delete(context.Background(), k.Namespace, metav1.DeleteOptions{})
	c.Kubectl.CoreV1().Namespaces().Delete(context.Background(), "cf-operator", metav1.DeleteOptions{})
	c.Kubectl.CoreV1().Namespaces().Delete(context.Background(), "eirini", metav1.DeleteOptions{})

	return nil
}

func (k KubeCF) GetPassword(c kubernetes.Cluster) (string, error) {
	secret, err := c.Kubectl.CoreV1().Secrets(k.Namespace).Get(context.TODO(), "var-cf-admin-password", metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "couldn't find password secret")
	}
	return string(secret.Data["password"]), nil
}

func (k KubeCF) genHelmSettings(c kubernetes.Cluster) []string {
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
			if !k.LB { // IF a LB won't assign IP addresses, we will forcefully assign those
				for i, ip := range c.GetPlatform().ExternalIPs() {
					helmArgs = append(helmArgs, "--set services."+s+".externalIPs["+strconv.Itoa(i)+"]="+ip)
				}
			}
		}
	} else {
		helmArgs = append(helmArgs, "--set features.ingress.enabled=true")
	}

	if k.Autoscaler {
		helmArgs = append(helmArgs, "--set features.autoscaler.enabled=true")
	}
	return helmArgs
}

func (k KubeCF) applyOperator(c kubernetes.Cluster, upgrade bool) error {
	currentdir, _ := os.Getwd()
	action := "install"
	if upgrade {
		action = "upgrade"
	}

	out, err := helpers.RunProc("helm "+action+" cf-operator --create-namespace --namespace cf-operator --wait "+k.QuarksOperator+" --set global.singleNamespace.name="+k.Namespace, currentdir)
	fmt.Println(out)
	if err != nil {
		return errors.New("Failed installing cf-operator")
	}

	if err := c.WaitForPodBySelectorRunning("cf-operator", "", 900); err != nil {
		return errors.Wrap(err, "failed waiting")
	}
	emoji.Println(":heavy_check_mark: Quarks Operator deployed correctly to the :rainbow: :cloud:")

	return nil
}

func (k KubeCF) applyKubeCF(c kubernetes.Cluster, upgrade bool) error {
	currentdir, _ := os.Getwd()

	// Setup KubeCF helm values
	helmArgs := k.genHelmSettings(c)

	action := "install"
	if upgrade {
		action = "upgrade"
	}

	_, err := helpers.RunProc("helm "+action+" kubecf --namespace "+k.Namespace+" "+k.ChartURL+" "+strings.Join(helmArgs, " "), currentdir)
	if err != nil {
		return errors.New("Failed installing kubecf")
	}

	// Wait for components to be up
	for _, s := range []string{"api", "nats", "cc-worker", "doppler"} {
		err = c.WaitUntilPodBySelectorExist(k.Namespace, "quarks.cloudfoundry.org/quarks-statefulset-name="+s, k.Timeout)
		if err != nil {
			return errors.Wrap(err, "Failed waiting for api")
		}
	}

	err = c.WaitForPodBySelectorRunning(k.Namespace, "app.kubernetes.io/name=kubecf", k.Timeout)
	if err != nil {
		return errors.Wrap(err, "failed waiting for kubecf to be ready")
	}
	emoji.Println(":heavy_check_mark: KubeCF deployed correctly to the :rainbow: :cloud:")
	return nil
}

func (k KubeCF) Deploy(c kubernetes.Cluster) error {
	emoji.Println(":ship: Deploying Quarks Operator")
	_, err := c.Kubectl.CoreV1().Namespaces().Get(
		context.Background(),
		"cf-operator",
		metav1.GetOptions{},
	)
	if err == nil {
		return errors.New("Namespace 'cf-operator' present already, run 'kubecfctl delete " + k.Version + "' first")
	}

	if err := k.applyOperator(c, false); err != nil {
		return errors.Wrap(err, "while deploying quarks operator")
	}

	emoji.Println(":ship: Deploying kubecf")

	if err := k.applyKubeCF(c, false); err != nil {
		return errors.Wrap(err, "while deploying quarks operator")
	}

	pwd, err := k.GetPassword(c)
	if err != nil {
		return errors.Wrap(err, "couldn't find password")
	}

	emoji.Println(":lock: CF Deployment ready, now you can login with: cf login --skip-ssl-validation -a https://api." + k.domain + " -u admin -p " + string(pwd))
	return nil
}

func (k KubeCF) Upgrade(c kubernetes.Cluster) error {
	emoji.Println(":ship: Upgrading Quarks Operator")
	_, err := c.Kubectl.CoreV1().Namespaces().Get(
		context.Background(),
		"cf-operator",
		metav1.GetOptions{},
	)
	if err != nil {
		return errors.New("Namespace 'cf-operator' not present")
	}

	if err := k.applyOperator(c, true); err != nil {
		return errors.Wrap(err, "while deploying quarks operator")
	}
	emoji.Println(":ship: Upgrading kubecf")

	if err := k.applyKubeCF(c, true); err != nil {
		return errors.Wrap(err, "while deploying quarks operator")
	}

	pwd, err := k.GetPassword(c)
	if err != nil {
		return errors.Wrap(err, "couldn't find password")
	}

	emoji.Println(":lock: CF Deployment ready, now you can login with: cf login --skip-ssl-validation -a https://api." + k.domain + " -u admin -p " + string(pwd))
	return nil
}
