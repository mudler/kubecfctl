package deployments

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/kyokomi/emoji"
	"github.com/mudler/kubecfctl/pkg/helpers"
	"github.com/mudler/kubecfctl/pkg/kubernetes"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type KubeCF struct {
	Version       string
	ChartURL      string
	quarksVersion string
	Namespace     string
	StorageClass  string
	domain        string
	Debug         bool

	AdditionalNamespaces []string

	Eirini, Ingress, Autoscaler, LB bool
	Timeout                         int
}

func (k *KubeCF) SetDomain(d string) {
	k.domain = d
}

func (k KubeCF) GetDomain() string {
	return k.domain
}

func (k KubeCF) GetVersion() string {
	return k.Version
}

func (k KubeCF) Describe() string {
	return emoji.Sprintf(":cloud: KubeCF version: %s\n:clipboard:Quarks version: %s\n:clipboard:KubeCF chart: %s", k.Version, k.quarksVersion, k.ChartURL)
}

func (k KubeCF) Delete(c kubernetes.Cluster) error {
	currentdir, _ := os.Getwd()

	quarks, err := GlobalCatalog.GetQuarks(k.quarksVersion)
	if err != nil {
		return err
	}
	err = quarks.Delete(c)
	if err != nil {
		return err
	}

	for _, ns := range k.AdditionalNamespaces {
		c.Kubectl.CoreV1().Namespaces().Delete(context.Background(), ns, metav1.DeleteOptions{})
		c.Kubectl.CoreV1().Namespaces().Delete(context.Background(), ns+"-eirini", metav1.DeleteOptions{})
	}

	c.Kubectl.CoreV1().Namespaces().Delete(context.Background(), k.Namespace, metav1.DeleteOptions{})
	c.Kubectl.CoreV1().Namespaces().Delete(context.Background(), k.Namespace+"-eirini", metav1.DeleteOptions{})

	helpers.RunProc("kubectl delete psp kubecf-default", currentdir, k.Debug)
	// workaround for: https://github.com/cloudfoundry-incubator/kubecf/issues/1582
	helpers.RunProc("kubectl delete clusterrolebinding eirini-cluster-rolebinding", currentdir, k.Debug)
	helpers.RunProc("kubectl delete clusterrole eirini-cluster-role", currentdir, k.Debug)
	emoji.Println(":heavy_check_mark: KubeCF deleted")

	return nil
}

func (k KubeCF) GetPassword(namespace string, c kubernetes.Cluster) (string, error) {
	secret, err := c.Kubectl.CoreV1().Secrets(namespace).Get(context.TODO(), "var-cf-admin-password", metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "couldn't find password secret")
	}
	return string(secret.Data["password"]), nil
}

func (k KubeCF) genHelmSettings(c kubernetes.Cluster, domain, ns string) []string {
	var helmArgs []string
	helmArgs = append(helmArgs, "--set system_domain="+domain)

	if k.Eirini {
		helmArgs = append(helmArgs, "--set features.eirini.enabled=true")
		helmArgs = append(helmArgs, "--set install_stacks[0]=sle15")
		helmArgs = append(helmArgs, "--set eirini.opi.namespace="+ns+"-eirini")
	}

	if len(k.StorageClass) != 0 {
		helmArgs = append(helmArgs, "--set kube.storage_class="+k.StorageClass)
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

func (k KubeCF) applyKubeCF(namespace, domain string, c kubernetes.Cluster, upgrade, psp bool) error {
	currentdir, _ := os.Getwd()

	// Setup KubeCF helm values
	helmArgs := k.genHelmSettings(c, domain, namespace)

	action := "install"
	if upgrade {
		action = "upgrade"
	}
	if !psp {
		helmArgs = append(helmArgs, "--set kube.psp.default=kubecf-default")
	}

	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond) // Build our new spinner
	s.Start()                                                    // Start the spinner
	out, err := helpers.RunProc("helm "+action+" kubecf --namespace "+namespace+" "+k.ChartURL+" "+strings.Join(helmArgs, " "), currentdir, k.Debug)
	if err != nil {
		fmt.Println(string(out))
		return errors.New("Failed installing kubecf")
	}

	s.Stop()
	// Wait for components to be up
	for _, s := range []string{"api", "nats", "cc-worker", "doppler"} {
		err = c.WaitUntilPodBySelectorExist(namespace, "quarks.cloudfoundry.org/quarks-statefulset-name="+s, k.Timeout)
		if err != nil {
			return errors.Wrap(err, "Failed waiting for api")
		}
	}

	err = c.WaitForPodBySelectorRunning(namespace, "app.kubernetes.io/name=kubecf", k.Timeout)
	if err != nil {
		return errors.Wrap(err, "failed waiting for kubecf to be ready")
	}
	emoji.Println(":heavy_check_mark: KubeCF deployed correctly to the :rainbow: :cloud:")
	return nil
}

func (k KubeCF) Deploy(c kubernetes.Cluster) error {
	currentdir, _ := os.Getwd()

	_, err := c.Kubectl.CoreV1().Namespaces().Get(
		context.Background(),
		"cf-operator",
		metav1.GetOptions{},
	)
	if err != nil {
		quarks, err := GlobalCatalog.GetQuarks(k.quarksVersion)
		if err != nil {
			return err
		}

		quarks.AdditionalNamespaces = k.AdditionalNamespaces
		err = quarks.Deploy(c)
		if err != nil {
			return err
		}
	} else {
		emoji.Println(":ship:Quarks operator already present. Delete if you want to test cleanly")
	}

	if k.Ingress {
		_, err = c.Kubectl.CoreV1().Namespaces().Get(
			context.Background(),
			"nginx-ingress",
			metav1.GetOptions{},
		)
		if err != nil {
			nginx, err := GlobalCatalog.GetNginx("3.7.1")
			if err != nil {
				return err
			}

			err = nginx.Deploy(c)
			if err != nil {
				return err
			}
		} else {
			emoji.Println(":ship:Nginx already present. Delete if you want to test cleanly")
		}
	}

	emoji.Println(":ship:Deploying kubecf")

	if err := k.applyKubeCF(k.Namespace, k.domain, c, false, true); err != nil {
		return errors.Wrap(err, "while deploying kubecf")
	}

	pwd, err := k.GetPassword(k.Namespace, c)
	if err != nil {
		return errors.Wrap(err, "couldn't find password")
	}
	// workaround for: https://github.com/cloudfoundry-incubator/kubecf/issues/1582
	if !k.Eirini {
		helpers.RunProc("kubectl delete clusterrolebinding eirini-cluster-rolebinding", currentdir, k.Debug)
		helpers.RunProc("kubectl delete clusterrole eirini-cluster-role", currentdir, k.Debug)
	}

	for _, ns := range k.AdditionalNamespaces {

		if k.Eirini {
			for _, psp := range []string{
				"bits-service", "eirini",
				"eirini-events", "eirini-metrics",
				"eirini-routing", "eirini-staging-reporter", "kubecf-eirini-app-psp",
			} {
				helpers.RunProc("kubectl delete psp "+psp, currentdir, k.Debug)

			}
			helpers.RunProc("kubectl delete clusterrole eirini-nodes-policy", currentdir, k.Debug)

			helpers.RunProc("kubectl delete clusterrolebinding eirini-cluster-rolebinding", currentdir, k.Debug)
			helpers.RunProc("kubectl delete clusterrole eirini-cluster-role", currentdir, k.Debug)
		}

		if err := k.applyKubeCF(ns, ns+"."+k.domain, c, false, false); err != nil {
			return errors.Wrap(err, "while deploying kubecf for namespace "+ns)
		}
		pwd, err := k.GetPassword(ns, c)
		if err != nil {
			return errors.Wrap(err, "couldn't find password")
		}
		// workaround for: https://github.com/cloudfoundry-incubator/kubecf/issues/1582
		if !k.Eirini {
			helpers.RunProc("kubectl delete clusterrolebinding eirini-cluster-rolebinding", currentdir, k.Debug)
			helpers.RunProc("kubectl delete clusterrole eirini-cluster-role", currentdir, k.Debug)
		}

		emoji.Println(":lock: " + ns + " CF Deployment ready, now you can login with: cf login --skip-ssl-validation -a https://api." + ns + "." + k.domain + " -u admin -p " + string(pwd))
	}

	emoji.Println(":lock:CF Deployment ready, now you can login with: cf login --skip-ssl-validation -a https://api." + k.domain + " -u admin -p " + string(pwd))
	return nil
}

func (k KubeCF) Upgrade(c kubernetes.Cluster) error {
	emoji.Println(":ship:Upgrading Quarks Operator")
	_, err := c.Kubectl.CoreV1().Namespaces().Get(
		context.Background(),
		"cf-operator",
		metav1.GetOptions{},
	)
	if err != nil {
		return errors.New("Namespace 'cf-operator' not present")
	}

	quarks, err := GlobalCatalog.GetQuarks(k.quarksVersion)
	if err != nil {
		return err
	}
	quarks.AdditionalNamespaces = k.AdditionalNamespaces
	err = quarks.Upgrade(c)
	if err != nil {
		return err
	}
	emoji.Println(":ship:Upgrading kubecf")

	if err := k.applyKubeCF(k.Namespace, k.domain, c, true, true); err != nil {
		return errors.Wrap(err, "while upgrading kubecf")
	}

	for _, ns := range k.AdditionalNamespaces {
		if err := k.applyKubeCF(ns, ns+"."+k.domain, c, true, false); err != nil {
			return errors.Wrap(err, "while upgrading kubecf for namespace "+ns)
		}
	}
	return nil
}
