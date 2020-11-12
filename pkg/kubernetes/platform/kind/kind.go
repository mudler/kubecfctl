package kind

import (
	"context"
	"strings"

	"github.com/kyokomi/emoji"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type kind struct {
	InternalIPs []string
}

func (k *kind) Describe() string {
	return emoji.Sprintf(":anchor:Detected kubernetes platform: %s\n:earth_americas:ExternalIPs: %s\n:curly_loop:InternalIPs: %s", k.String(), k.ExternalIPs(), k.InternalIPs)
}

func (k *kind) String() string { return "kind" }

func (k *kind) Detect(kube *kubernetes.Clientset) bool {
	nodes, err := kube.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return false
	}
	for _, n := range nodes.Items {
		if strings.Contains(n.Spec.ProviderID, "kind://") {
			return true
		}
	}
	return false
}

func (k *kind) Load(kube *kubernetes.Clientset) error {
	nodes, err := kube.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	// See also https://github.com/kubernetes/kubernetes/blob/47943d5f9ce7dbe8fbf805ff76a5eb9726c6af0c/test/e2e/framework/util.go#L1266
	internalIPs := []string{}
	for _, n := range nodes.Items {
		for _, address := range n.Status.Addresses {
			if address.Type == "InternalIP" {
				internalIPs = append(internalIPs, address.Address)
			}
		}
	}
	k.InternalIPs = internalIPs
	return nil
}

func (k *kind) ExternalIPs() []string {
	return k.InternalIPs
}

func NewPlatform() *kind {
	return &kind{}
}
