package deployments

import (
	"strings"

	kubernetes "github.com/mudler/kubecfctl/pkg/kubernetes"

	"github.com/pkg/errors"
)

type Catalog struct {
	Nginx   []NginxIngress
	KubeCF  []KubeCF
	Stratos []Stratos
}

var GlobalCatalog = Catalog{
	Stratos: []Stratos{
		{
			Version:   "4.2.1",
			ChartURL:  "https://github.com/cloudfoundry/stratos/releases/download/4.2.1/console-helm-chart-4.2.1-15dcb83ab.tgz",
			Namespace: "stratos",
		},
	},
	Nginx: []NginxIngress{
		{
			Version:   "3.7.1",
			ChartURL:  "https://github.com/kubernetes/ingress-nginx/releases/download/ingress-nginx-3.7.1/ingress-nginx-3.7.1.tgz",
			Namespace: "nginx-ingress",
		},
	},
	KubeCF: []KubeCF{
		{
			Version:        "2.6.1",
			ChartURL:       "https://github.com/cloudfoundry-incubator/kubecf/releases/download/v2.6.1/kubecf-v2.6.1.tgz",
			Namespace:      "kubecf",
			QuarksOperator: "https://s3.amazonaws.com/cf-operators/release/helm-charts/cf-operator-6.1.17%2B0.gec409fd7.tgz",
		},
		{
			Version:        "2.5.8",
			ChartURL:       "https://github.com/cloudfoundry-incubator/kubecf/releases/download/v2.5.8/kubecf-v2.5.8.tgz",
			Namespace:      "kubecf",
			QuarksOperator: "https://github.com/cloudfoundry-incubator/quarks-operator/releases/download/v6.1.17/cf-operator-6.1.17+0.gec409fd7.tgz",
		},
	},
}

func (c Catalog) GetKubeCF(version string) (KubeCF, error) {
	for _, r := range c.KubeCF {
		if r.Version == version {
			return r, nil
		}
	}
	return KubeCF{}, errors.New("No version found")
}

func (c Catalog) GetNginx(version string) (NginxIngress, error) {
	for _, r := range c.Nginx {
		if r.Version == version {
			return r, nil
		}
	}
	return NginxIngress{}, errors.New("No version found")
}

func (c Catalog) GetStratos(version string) (Stratos, error) {
	for _, r := range c.Stratos {
		if r.Version == version {
			return r, nil
		}
	}
	return Stratos{}, errors.New("No version found")
}

func (c Catalog) GetList() []interface{} {
	var res []interface{}
	for _, d := range c.KubeCF {
		res = append(res, []interface{}{"kubecf", d.Version})
	}
	for _, d := range c.Nginx {
		res = append(res, []interface{}{"nginx-ingress", d.Version})
	}
	for _, d := range c.Stratos {
		res = append(res, []interface{}{"stratos", d.Version})
	}
	return res
}

func (c Catalog) Search(term string) []interface{} {
	var res []interface{}
	for _, d := range c.KubeCF {
		if strings.Contains(d.Version, term) || strings.Contains("kubecf", term) {
			res = append(res, []interface{}{"kubecf", d.Version})
		}
	}
	for _, d := range c.Nginx {
		if strings.Contains(d.Version, term) || strings.Contains("nginx-ingress", term) {
			res = append(res, []interface{}{"nginx-ingress", d.Version})
		}
	}
	for _, d := range c.Stratos {
		if strings.Contains(d.Version, term) || strings.Contains("stratos", term) {
			res = append(res, []interface{}{"stratos", d.Version})
		}
	}
	return res
}

type DeploymentOptions struct {
	Eirini              bool
	Timeout             int
	Ingress             bool
	Debug               bool
	Version             string
	ChartURL, QuarksURL string
}

func (c Catalog) Deployment(name string, opts DeploymentOptions) (kubernetes.Deployment, error) {
	switch name {
	case "kubecf":
		if opts.ChartURL != "" || opts.QuarksURL != "" { // Return custom version specified
			return &KubeCF{
				Version:        "Custom",
				ChartURL:       opts.ChartURL,
				Namespace:      "kubecf",
				QuarksOperator: opts.QuarksURL,
				Eirini:         opts.Eirini,
				Timeout:        opts.Timeout,
				Ingress:        opts.Ingress,
				Debug:          opts.Debug,
			}, nil
		}
		if len(opts.Version) == 0 { // Get default version if not specified
			kubecf, err := c.GetKubeCF("2.6.1")
			if err != nil {
				return nil, err
			}
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
