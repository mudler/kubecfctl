package deployments

import (
	"strings"

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
