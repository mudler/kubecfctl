package kubernetes

import (
	"errors"
	"fmt"
)

type Installer struct {
}

type Deployment interface {
	Deploy(Cluster) error
	Upgrade(Cluster) error
	SetDomain(d string)
	GetDomain() string
	Delete(Cluster) error
	Describe() string
}

func NewInstaller() *Installer {
	return &Installer{}
}

func (i *Installer) Install(d Deployment, cluster Cluster) error {
	fmt.Println(d.Describe())

	// Automatically set a deployment domain based on platform reported ExternalIPs
	if d.GetDomain() == "" {
		ips := cluster.GetPlatform().ExternalIPs()
		if len(ips) == 0 {
			return errors.New("Could not detect cluster ExternalIPs and no deployment domain was specified")
		}
		d.SetDomain(fmt.Sprintf("%s.nip.io", ips[0]))
	}
	return d.Deploy(cluster)
}

func (i *Installer) Delete(d Deployment, cluster Cluster) error {
	return d.Delete(cluster)
}

func (i *Installer) Upgrade(d Deployment, cluster Cluster) error {
	return d.Upgrade(cluster)
}
