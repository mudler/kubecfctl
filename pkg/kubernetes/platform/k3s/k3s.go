package k3s

import (
	"context"
	"strings"

	"github.com/kyokomi/emoji"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type k3s struct {
	InternalIPs, externalIPs []string
}

func (k *k3s) Describe() string {
	return emoji.Sprintf(":anchor: Detected kubernetes platform: %s\nExternalIPs: %s\nInternalIPs: %s\n", k.String(), k.ExternalIPs(), k.InternalIPs)
}

func (k *k3s) String() string { return "k3s" }

func (k *k3s) Detect(kube *kubernetes.Clientset) bool {
	nodes, err := kube.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return false
	}
	for _, n := range nodes.Items {
		if strings.Contains(n.Spec.ProviderID, "k3s://") {
			return true
		}
	}
	return false
}

func (k *k3s) Load(kube *kubernetes.Clientset) error {
	nodes, err := kube.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	// See also https://github.com/kubernetes/kubernetes/blob/47943d5f9ce7dbe8fbf805ff76a5eb9726c6af0c/test/e2e/framework/util.go#L1266
	internalIPs := []string{}
	externalIPs := []string{}
	for _, n := range nodes.Items {
		for _, address := range n.Status.Addresses {
			switch address.Type {
			case "InternalIP":
				internalIPs = append(internalIPs, address.Address)
			case "ExternalIP":
				externalIPs = append(externalIPs, address.Address)
			}
		}
	}
	k.InternalIPs = internalIPs
	k.externalIPs = externalIPs

	return nil
}

func (k *k3s) ExternalIPs() []string {
	return k.externalIPs
}

func NewPlatform() *k3s {
	return &k3s{}
}
