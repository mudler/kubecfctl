package deployments

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/kyokomi/emoji"
	"github.com/mudler/kubecfctl/pkg/helpers"
	"github.com/mudler/kubecfctl/pkg/kubernetes"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
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

	ccdbEncKey, currentKey string
	encKeys                map[string]string

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

	if len(k.ccdbEncKey) != 0 {
		helmArgs = append(helmArgs, "--set credentials.cc_db_encryption_key="+k.ccdbEncKey)
	}
	if len(k.encKeys) != 0 {
		i := 0
		for label, key := range k.encKeys {
			helmArgs = append(helmArgs, "--set ccdb.encryption.rotation.key_labels["+strconv.Itoa(i)+"]="+label)
			helmArgs = append(helmArgs, "--set credentials.ccdb_key_label_"+label+"="+key)
			i++
		}
		helmArgs = append(helmArgs, "--set credentials.cc_db_encryption_key="+k.ccdbEncKey)
	}

	if len(k.currentKey) != 0 {
		helmArgs = append(helmArgs, "--set ccdb.encryption.rotation.current_key_label="+k.currentKey)
	}
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

// db_encryption_key: EzmCLwwF6eV0QxyjrRD4w3QkNaVzQO4echeHyLzNMqoQ8cGiNt2CDpPIxWpYPz8i
// database_encryption:
//   keys: {"encryption_key_0":"rMNnJcQ8Gb8DJc9hkEuICJOOgTJrc8lSfMoOCA5sRQIeYsMFfI5XqMvcJhZKFeUZ"}
//   current_key_label: "encryption_key_0"
//   pbkdf2_hmac_iterations: 2048

type ccEnc struct {
	Current string `yaml:"current_key_label"`
	Keys    string `yaml:"keys"`
}
type ccConfig struct {
	Encryption ccEnc  `yaml:"database_encryption"`
	DbKey      string `yaml:"db_encryption_key"`
}

func (k KubeCF) Restore(c kubernetes.Cluster, output string) error {

	err := k.Deploy(c)
	if err != nil {
		return errors.Wrap(err, "while deploying kubecf")
	}

	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond) // Build our new spinner
	s.Start()                                                    // Start the spinner
	defer s.Stop()
	s.Suffix = " Extracting encryption configuration"

	dat, err := ioutil.ReadFile(filepath.Join(output, "cc_config.yaml"))
	if err != nil {
		return errors.Wrap(err, "while reading cc_config.yaml")
	}
	config := ccConfig{}
	err = yaml.Unmarshal(dat, &config)
	if err != nil {
		return errors.Wrap(err, "while unmarshalling cc_config.yaml")
	}

	var keys map[string]string

	err = json.Unmarshal([]byte(config.Encryption.Keys), &keys)
	if err != nil {
		return errors.Wrap(err, "while unmarshalling encryption keys")
	}
	k.encKeys = keys
	k.ccdbEncKey = config.DbKey
	k.currentKey = config.Encryption.Current

	s.Suffix = " Disable db restrictions"
	out, stderr, err := c.Exec(k.Namespace, "database-0", "database", "mysql", `SET GLOBAL pxc_strict_mode=PERMISSIVE;
SET GLOBAL
sql_mode='STRICT_ALL_TABLES,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION';
set GLOBAL innodb_strict_mode='OFF';
quit;
`)
	if err != nil {
		fmt.Println(out)
		fmt.Println(stderr)
		return errors.Wrap(err, "while disabling db restrictions")
	}

	dat, err = ioutil.ReadFile(filepath.Join(output, "uaadb-src.sql"))
	if err != nil {
		return errors.Wrap(err, "while reading up uaa backup")
	}
	s.Suffix = " Restoring UAA"
	out, stderr, err = c.Exec(k.Namespace, "database-0", "database", "mysql uaa", string(dat))
	if err != nil {
		fmt.Println(out)
		fmt.Println(stderr)
		return errors.Wrap(err, "while backing up uaa db")
	}

	s.Suffix = " Restoring Blobstore"

	_, err = helpers.RunProcNoErr("kubectl exec --namespace "+k.Namespace+" singleton-blobstore-0 -- tar xfz - -C < blob.tgz", output, k.Debug)
	if err != nil {
		return errors.Wrap(err, "while restoring up blobstore")
	}
	_, err = helpers.RunProcNoErr("kubectl delete pod --namespace "+k.Namespace+" singleton-blobstore-0", output, k.Debug)
	if err != nil {
		return errors.Wrap(err, "while restarting blobstore")
	}

	s.Suffix = " Restoring CCDB"

	out, stderr, err = c.Exec(k.Namespace, "database-0", "database", "mysql", `drop database cloud_controller; 
create database	cloud_controller;
quit;
`)
	if err != nil {
		fmt.Println(out)
		fmt.Println(stderr)
		return errors.Wrap(err, "while pruning cc db")
	}

	dat, err = ioutil.ReadFile(filepath.Join(output, "ccdb-src.sql"))
	if err != nil {
		return errors.Wrap(err, "while reading up ccdb backup")
	}
	out, stderr, err = c.Exec(k.Namespace, "database-0", "database", "mysql cloud_controller", string(dat))
	if err != nil {
		fmt.Println(out)
		fmt.Println(stderr)
		return errors.Wrap(err, "while backing up ccdb db")
	}

	err = k.Upgrade(c)
	if err != nil {
		return errors.Wrap(err, "while deploying kubecf")
	}
	return nil

}

func (k KubeCF) Backup(c kubernetes.Cluster, output string) error {
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond) // Build our new spinner
	s.Start()                                                    // Start the spinner
	defer s.Stop()

	s.Suffix = " Backing up blobstore"
	_, err := helpers.RunProcNoErr("kubectl exec --namespace "+k.Namespace+" singleton-blobstore-0 -- tar cfz - --exclude=/var/vcap/store/shared/tmp /var/vcap/store/shared > blob.tgz", output, k.Debug)
	if err != nil {
		return errors.Wrap(err, "while backing up blobstore")
	}

	s.Suffix = " Disable db restrictions"
	out, stderr, err := c.Exec(k.Namespace, "database-0", "database", "mysql", `SET GLOBAL pxc_strict_mode=PERMISSIVE;
SET GLOBAL
sql_mode='STRICT_ALL_TABLES,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION';
set GLOBAL innodb_strict_mode='OFF';
quit;
`)
	if err != nil {
		fmt.Println(out)
		fmt.Println(stderr)
		return errors.Wrap(err, "while disabling db restrictions")
	}

	s.Suffix = " Backing up uaa"
	out, stderr, err = c.Exec(k.Namespace, "database-0", "database", "mysqldump uaa", "")
	if err != nil {
		fmt.Println(out)
		fmt.Println(stderr)
		return errors.Wrap(err, "while backing up uaa db")
	}
	err = ioutil.WriteFile(filepath.Join(output, "uaadb-src.sql"), []byte(out), 0644)
	if err != nil {
		fmt.Println(stderr)
		return errors.Wrap(err, "while backing up uaa db")
	}

	s.Suffix = " Backing up ccdb"
	out, stderr, err = c.Exec(k.Namespace, "database-0", "database", "mysqldump cloud_controller", "")
	if err != nil {
		fmt.Println(stderr)
		return errors.Wrap(err, "while backing up ccdb db")
	}
	err = ioutil.WriteFile(filepath.Join(output, "ccdb-src.sql"), []byte(out), 0644)
	if err != nil {
		return errors.Wrap(err, "while backing up ccdb db")
	}

	s.Suffix = " Backing up cloud_controller_ng.yml"
	out, stderr, err = c.Exec(k.Namespace, "api-0", "api", "cat /var/vcap/jobs/cloud_controller_ng/config/cloud_controller_ng.yml", "")
	if err != nil {
		fmt.Println(stderr)
		return errors.Wrap(err, "while backing up cc config")
	}
	err = ioutil.WriteFile(filepath.Join(output, "cc_config.yaml"), []byte(out), 0644)
	if err != nil {
		fmt.Println(stderr)
		return errors.Wrap(err, "while backing up cc config")
	}

	return nil
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
