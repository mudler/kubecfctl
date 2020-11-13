package deployments

import (
	"strings"

	kubernetes "github.com/mudler/kubecfctl/pkg/kubernetes"

	"github.com/pkg/errors"
)

type Deployment interface {
	GetVersion() string
}

type catalogVersions map[string]map[string]kubernetes.Deployment

type available map[string]kubernetes.Deployment

type Catalog map[string]available

var GlobalCatalog = Catalog{

	"kubecf": available{
		"2.6.1": &KubeCF{
			Version:       "2.6.1",
			ChartURL:      "https://github.com/cloudfoundry-incubator/kubecf/releases/download/v2.6.1/kubecf-v2.6.1.tgz",
			Namespace:     "kubecf",
			quarksVersion: "6.1.17",
		},
		"2.5.8": &KubeCF{
			Version:       "2.5.8",
			ChartURL:      "https://github.com/cloudfoundry-incubator/kubecf/releases/download/v2.5.8/kubecf-v2.5.8.tgz",
			Namespace:     "kubecf",
			quarksVersion: "6.1.17",
		},
	},

	"stratos": available{
		"4.2.1": &Stratos{
			Version:   "4.2.1",
			ChartURL:  "https://github.com/cloudfoundry/stratos/releases/download/4.2.1/console-helm-chart-4.2.1-15dcb83ab.tgz",
			Namespace: "stratos",
		},
	},

	"nginx": available{
		"3.7.1": &NginxIngress{
			Version:   "3.7.1",
			ChartURL:  "https://github.com/kubernetes/ingress-nginx/releases/download/ingress-nginx-3.7.1/ingress-nginx-3.7.1.tgz",
			Namespace: "nginx-ingress",
		},
	},

	"quarks": available{
		"6.1.17": &Quarks{
			Version:   "6.1.17",
			Namespace: "kubecf",
			ChartURL:  "https://s3.amazonaws.com/cf-operators/release/helm-charts/cf-operator-6.1.17%2B0.gec409fd7.tgz",
		},
	},

	"carrier": available{
		"master": &Carrier{
			Version:       "master",
			ChartURL:      "https://github.com/SUSE/carrier",
			quarksVersion: "6.1.17",
		},
	},
}

func (c Catalog) GetKubeCF(version string) (KubeCF, error) {
	d, ok := c["kubecf"][version]
	if !ok {
		return KubeCF{}, errors.New("version not found")
	}
	return *(d.(*KubeCF)), nil
}

func (c Catalog) GetCarrier(version string) (Carrier, error) {
	d, ok := c["carrier"][version]
	if !ok {
		return Carrier{}, errors.New("version not found")
	}
	return *(d.(*Carrier)), nil
}

func (c Catalog) GetQuarks(version string) (Quarks, error) {
	d, ok := c["quarks"][version]
	if !ok {
		return Quarks{}, errors.New("version not found")
	}
	return *(d.(*Quarks)), nil
}

func (c Catalog) GetNginx(version string) (NginxIngress, error) {
	d, ok := c["nginx"][version]
	if !ok {
		return NginxIngress{}, errors.New("version not found")
	}
	return *(d.(*NginxIngress)), nil
}

func (c Catalog) GetStratos(version string) (Stratos, error) {
	d, ok := c["stratos"][version]
	if !ok {
		return Stratos{}, errors.New("version not found")
	}
	return *(d.(*Stratos)), nil
}

func (c Catalog) GetList() []interface{} {
	var res []interface{}
	for p, s := range c {
		for v := range s {
			res = append(res, []interface{}{p, v})
		}
	}

	return res
}

func (c Catalog) Search(term string) []interface{} {
	var res []interface{}

	for p, s := range c {
		for v := range s {
			if strings.Contains(v, term) || strings.Contains(p, term) {
				res = append(res, []interface{}{p, v})
			}
		}
	}

	return res
}

type DeploymentOptions struct {
	Eirini                             bool
	Timeout                            int
	Ingress                            bool
	Debug                              bool
	Version                            string
	ChartURL, QuarksURL                string
	AdditionalNamespaces               []string
	RegistryUsername, RegistryPassword string
}

func (c Catalog) Deployment(name string, opts DeploymentOptions) (kubernetes.Deployment, error) {
	switch name {
	case "kubecf":
		if opts.ChartURL != "" || opts.QuarksURL != "" { // Return custom version specified
			return &KubeCF{
				Version:              "Custom",
				ChartURL:             opts.ChartURL,
				Namespace:            "kubecf",
				quarksVersion:        opts.QuarksURL,
				Eirini:               opts.Eirini,
				Timeout:              opts.Timeout,
				Ingress:              opts.Ingress,
				Debug:                opts.Debug,
				AdditionalNamespaces: opts.AdditionalNamespaces,
			}, nil
		}
		if len(opts.Version) == 0 { // Get default version if not specified
			kubecf, err := c.GetKubeCF("2.6.1")
			if err != nil {
				return nil, err
			}
			kubecf.Eirini = opts.Eirini
			kubecf.Timeout = opts.Timeout
			kubecf.Ingress = opts.Ingress
			kubecf.Debug = opts.Debug
			kubecf.AdditionalNamespaces = opts.AdditionalNamespaces

			return &kubecf, nil
		}
		kubecf, err := c.GetKubeCF(opts.Version)
		if err != nil {
			return nil, err
		}
		kubecf.Eirini = opts.Eirini
		kubecf.Timeout = opts.Timeout
		kubecf.Ingress = opts.Ingress
		kubecf.Debug = opts.Debug
		kubecf.AdditionalNamespaces = opts.AdditionalNamespaces
		return &kubecf, nil
	case "nginx-ingress":
		if opts.ChartURL != "" {
			nginx := NginxIngress{
				Version:   "Custom",
				ChartURL:  opts.ChartURL,
				Namespace: "nginx-ingress",
				Debug:     opts.Debug,
			}
			return &nginx, nil
		}
		if len(opts.Version) == 0 {
			nginx, err := c.GetNginx("3.7.1")
			if err != nil {
				return nil, err
			}
			return &nginx, nil
		}
		nginx, err := c.GetNginx(opts.Version)
		if err != nil {
			return nil, err
		}
		nginx.Debug = opts.Debug
		return &nginx, nil
	case "quarks":
		if opts.ChartURL != "" {
			quarks := Quarks{
				Version:              "Custom",
				ChartURL:             opts.ChartURL,
				Debug:                opts.Debug,
				AdditionalNamespaces: opts.AdditionalNamespaces,
			}
			return &quarks, nil
		}
		if len(opts.Version) == 0 {
			quarks, err := c.GetQuarks("6.1.17")
			if err != nil {
				return nil, err
			}
			quarks.Debug = opts.Debug
			quarks.AdditionalNamespaces = opts.AdditionalNamespaces
			return &quarks, nil
		}
		quarks, err := c.GetQuarks(opts.Version)
		if err != nil {
			return nil, err
		}
		quarks.Debug = opts.Debug
		quarks.AdditionalNamespaces = opts.AdditionalNamespaces
		return &quarks, nil
	case "carrier":
		if opts.ChartURL != "" {
			carrier := Carrier{
				Version:          "Custom",
				ChartURL:         opts.ChartURL,
				Debug:            opts.Debug,
				RegistryUsername: opts.RegistryUsername,
				RegistryPassword: opts.RegistryPassword,
			}
			return &carrier, nil
		}
		if len(opts.Version) == 0 {
			carrier, err := c.GetCarrier("master")
			if err != nil {
				return nil, err
			}
			carrier.Debug = opts.Debug
			carrier.RegistryUsername = opts.RegistryUsername
			carrier.RegistryPassword = opts.RegistryPassword
			return &carrier, nil
		}
		carrier, err := c.GetCarrier(opts.Version)
		if err != nil {
			return nil, err
		}
		carrier.Debug = opts.Debug
		carrier.RegistryUsername = opts.RegistryUsername
		carrier.RegistryPassword = opts.RegistryPassword
		return &carrier, nil
	case "stratos":
		if opts.ChartURL != "" {
			stratos := Stratos{
				Version:   "Custom",
				ChartURL:  opts.ChartURL,
				Namespace: "stratos",
				Debug:     opts.Debug,
			}
			return &stratos, nil
		}
		if len(opts.Version) == 0 {
			stratos, err := c.GetStratos("4.2.1")
			if err != nil {
				return nil, err
			}
			return &stratos, nil
		}
		stratos, err := c.GetStratos(opts.Version)
		if err != nil {
			return nil, err
		}
		stratos.Debug = opts.Debug
		return &stratos, nil
	default:
		return nil, errors.New("Invalid deployment")
	}
}
