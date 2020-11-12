package deployments

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/kyokomi/emoji"
	"github.com/mudler/kubecfctl/pkg/helpers"
	"github.com/mudler/kubecfctl/pkg/kubernetes"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Quarks struct {
	Version              string
	ChartURL             string
	Namespace            string
	domain               string
	AdditionalNamespaces []string
	Debug                bool

	Timeout int
}

func (k *Quarks) SetDomain(d string) {
	k.domain = d
}

func (k Quarks) GetDomain() string {
	return k.domain
}

func (k Quarks) Describe() string {
	return emoji.Sprintf(":cloud:Quarks version: %s\n:clipboard:Quarks chart: %s", k.Version, k.ChartURL)
}

func (k Quarks) Delete(c kubernetes.Cluster) error {
	currentdir, _ := os.Getwd()

	helpers.RunProc("kubectl delete crds boshdeployments.quarks.cloudfoundry.org", currentdir, k.Debug)
	helpers.RunProc("kubectl delete crds quarksjobs.quarks.cloudfoundry.org", currentdir, k.Debug)
	helpers.RunProc("kubectl delete crds quarkssecrets.quarks.cloudfoundry.org", currentdir, k.Debug)
	helpers.RunProc("kubectl delete crds quarksstatefulsets.quarks.cloudfoundry.org", currentdir, k.Debug)

	if len(k.AdditionalNamespaces) != 0 {
		for _, ns := range k.AdditionalNamespaces {
			c.Kubectl.CoreV1().Namespaces().Delete(context.Background(), ns, metav1.DeleteOptions{})

		}
	}

	c.Kubectl.CoreV1().Namespaces().Delete(context.Background(), k.Namespace, metav1.DeleteOptions{})
	c.Kubectl.CoreV1().Namespaces().Delete(context.Background(), "cf-operator", metav1.DeleteOptions{})

	return nil
}

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func String(length int) string {
	return StringWithCharset(length, charset)
}

func (q Quarks) prepareAdditionalNamespace(c kubernetes.Cluster, namespace string) error {
	emoji.Println(":clipboard:Preparing namespace ", namespace)

	roleName := namespace + "cfo" + String(5)
	saName := namespace + "cfo" + String(5)

	_, err := c.Kubectl.CoreV1().Namespaces().Create(context.Background(),
		&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
				Labels: map[string]string{
					"quarks.cloudfoundry.org/qjob-service-account": saName,
					"quarks.cloudfoundry.org/monitored":            "cfo",
				},
			},
		},
		metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = c.Kubectl.CoreV1().ServiceAccounts(namespace).Create(context.Background(),
		&v1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: saName,
			},
		},
		metav1.CreateOptions{})
	if err != nil {
		return err
	}
	currentdir, _ := os.Getwd()

	out, err := helpers.RunProc(fmt.Sprintf("kubectl --namespace %s create rolebinding --clusterrole %s --serviceaccount %s %s", namespace, "qjob-persist-output", namespace+":"+saName, roleName), currentdir, q.Debug)
	if err != nil {
		fmt.Println(string(out))
		return err
	}
	fmt.Println(string(out))

	return nil
}

func (k Quarks) ApplyOperator(c kubernetes.Cluster, upgrade bool) error {
	currentdir, _ := os.Getwd()
	action := "install"
	if upgrade {
		action = "upgrade"
	}

	out, err := helpers.RunProc("helm "+action+" cf-operator --create-namespace --namespace cf-operator --wait "+k.ChartURL+" --set global.singleNamespace.name="+k.Namespace, currentdir, k.Debug)
	fmt.Println(out)
	if err != nil {
		return errors.New("Failed installing quarks-operator")
	}

	if err := c.WaitForPodBySelectorRunning("cf-operator", "", 900); err != nil {
		return errors.Wrap(err, "failed waiting")
	}

	if len(k.AdditionalNamespaces) != 0 && action == "install" {
		for _, ns := range k.AdditionalNamespaces {
			err := k.prepareAdditionalNamespace(c, ns)
			if err != nil {
				return errors.Wrap(err, "Failed preparing additional NS "+ns)
			}
		}
	}
	emoji.Println(":heavy_check_mark:Quarks Operator deployed correctly to the :rainbow: :cloud:")

	return nil
}

func (k Quarks) GetVersion() string {
	return k.Version
}

func (k Quarks) Deploy(c kubernetes.Cluster) error {
	emoji.Println(":ship: Deploying Quarks Operator")
	_, err := c.Kubectl.CoreV1().Namespaces().Get(
		context.Background(),
		"cf-operator",
		metav1.GetOptions{},
	)
	if err == nil {
		return errors.New("Namespace 'cf-operator' present already, run 'kubecfctl delete " + k.Version + "' first")
	}

	if err := k.ApplyOperator(c, false); err != nil {
		return errors.Wrap(err, "while deploying quarks operator")
	}

	return nil
}

func (k Quarks) Upgrade(c kubernetes.Cluster) error {
	emoji.Println(":ship:Upgrading Quarks Operator")
	_, err := c.Kubectl.CoreV1().Namespaces().Get(
		context.Background(),
		"cf-operator",
		metav1.GetOptions{},
	)
	if err != nil {
		return errors.New("Namespace 'cf-operator' not present")
	}

	if err := k.ApplyOperator(c, true); err != nil {
		return errors.Wrap(err, "while deploying quarks operator")
	}
	return nil
}
