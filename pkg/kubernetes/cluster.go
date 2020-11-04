package kubernetes

import (
	"context"
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/kyokomi/emoji"
	"github.com/pkg/errors"

	k3s "github.com/mudler/kubecfctl/pkg/kubernetes/platform/k3s"
	kind "github.com/mudler/kubecfctl/pkg/kubernetes/platform/kind"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Platform interface {
	Detect(*kubernetes.Clientset) bool
	Describe() string
	String() string
	Load(*kubernetes.Clientset) error
	ExternalIPs() []string
}

var SupportedPlatforms []Platform = []Platform{kind.NewPlatform(), k3s.NewPlatform()}

type Cluster struct {
	//	InternalIPs []string
	//	Ingress     bool
	Kubectl  *kubernetes.Clientset
	platform Platform
}

func NewCluster(kubeconfig string) (*Cluster, error) {
	c := &Cluster{}
	return c, c.Connect(kubeconfig)
}

func (c *Cluster) GetPlatform() Platform {
	return c.platform
}

func (c *Cluster) Connect(config string) error {
	restConfig, err := clientcmd.BuildConfigFromFlags("", config)
	if err != nil {
		return err
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	c.Kubectl = clientset
	c.detectPlatform()
	if c.platform == nil {
		return errors.New("No supported platform detected. Bailing out")
	}

	return c.platform.Load(clientset)
}

func (c *Cluster) detectPlatform() {
	for _, p := range SupportedPlatforms {
		if p.Detect(c.Kubectl) {
			c.platform = p
			return
		}
	}
}

// return a condition function that indicates whether the given pod is
// currently running
func (c *Cluster) isPodRunning(podName, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		pod, err := c.Kubectl.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		for _, cont := range pod.Status.ContainerStatuses {
			if cont.State.Waiting != nil {
				//fmt.Println("containers still in waiting")
				return false, nil
			}
		}

		for _, cont := range pod.Status.InitContainerStatuses {
			if cont.State.Waiting != nil || cont.State.Running != nil {
				return false, nil
			}
		}

		switch pod.Status.Phase {
		case v1.PodRunning, v1.PodSucceeded:
			return true, nil
		case v1.PodFailed:
			return false, nil
		}
		return false, nil
	}
}

func (c *Cluster) podExists(namespace, selector string) wait.ConditionFunc {
	return func() (bool, error) {
		podList, err := c.ListPods(namespace, selector)
		if err != nil {
			return false, err
		}
		if len(podList.Items) == 0 {
			return false, nil
		}
		return true, nil
	}
}

// Poll up to timeout seconds for pod to enter running state.
// Returns an error if the pod never enters the running state.
func (c *Cluster) WaitForPodRunning(namespace, podName string, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, c.isPodRunning(podName, namespace))
}

// ListPods returns the list of currently scheduled or running pods in `namespace` with the given selector
func (c *Cluster) ListPods(namespace, selector string) (*v1.PodList, error) {
	listOptions := metav1.ListOptions{}
	if len(selector) > 0 {
		listOptions.LabelSelector = selector
	}
	podList, err := c.Kubectl.CoreV1().Pods(namespace).List(context.Background(), listOptions)
	if err != nil {
		return nil, err
	}
	return podList, nil
}

// Wait up to timeout seconds for all pods in 'namespace' with given 'selector' to enter running state.
// Returns an error if no pods are found or not all discovered pods enter running state.
func (c *Cluster) WaitUntilPodBySelectorExist(namespace, selector string, timeout int) error {
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond) // Build our new spinner
	s.Start()                                                    // Start the spinner
	defer s.Stop()
	s.Suffix = emoji.Sprintf(" Waiting for resource %s to be created in %s ... :zzz: ", selector, namespace)
	return wait.PollImmediate(time.Second, time.Duration(timeout)*time.Second, c.podExists(namespace, selector))
}

// Wait up to timeout seconds for all pods in 'namespace' with given 'selector' to enter running state.
// Returns an error if no pods are found or not all discovered pods enter running state.
func (c *Cluster) WaitForPodBySelectorRunning(namespace, selector string, timeout int) error {
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond) // Build our new spinner
	s.Start()                                                    // Start the spinner
	defer s.Stop()
	s.Suffix = emoji.Sprintf(" Waiting for resource %s to be running in %s ... :zzz: ", selector, namespace)
	podList, err := c.ListPods(namespace, selector)
	if err != nil {
		return errors.Wrapf(err, "failed listingpods with selector %s", selector)
	}

	if len(podList.Items) == 0 {
		return fmt.Errorf("no pods in %s with selector %s", namespace, selector)
	}

	for _, pod := range podList.Items {
		s.Suffix = emoji.Sprintf(" Waiting for pod %s to be running in %s ... :zzz: ", pod.Name, namespace)
		if err := c.WaitForPodRunning(namespace, pod.Name, time.Duration(timeout)*time.Second); err != nil {
			return errors.Wrapf(err, "failed waiting for %s", pod.Name)
		}
	}
	return nil
}
