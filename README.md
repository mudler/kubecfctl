# kubecfctl
unofficial KubeCF CLI installer tool

[![asciicast](https://asciinema.org/a/B3b82mZ6XAvJAP7mbbXhOqkSQ.svg)](https://asciinema.org/a/B3b82mZ6XAvJAP7mbbXhOqkSQ)

## Prerequisite

- helm 3.x
- kubectl

## Install

You can find pre-compiled binaries for major platforms in [the release page](https://github.com/mudler/kubecfctl/releases).
Below you will find generic instructions to install kubecfctl in your system.

### Standalone binary

```bash
$ wget https://github.com/mudler/kubecfctl/releases/download/0.2.2/kubecfctl-0.2.2-linux-amd64 -O kubecfctl
$ chmod +x kubecfctl
$ mv kubecfctl /usr/local/bin
$ kubecfctl ...
```

### Kubectl plugin

You can also install kubecfctl as a Kubectl plugin. Just save the file as `kubectl-kubecf`
```bash
$ wget https://github.com/mudler/kubecfctl/releases/download/0.2.2/kubecfctl-0.2.2-linux-amd64 -O kubectl-kubecf
$ chmod +x kubectl-kubecf
$ mv kubectl-kubecf /usr/local/bin
$ kubectl kubecf ...
```

## Kubecfctl + K3s = :heart:

Kubecfctl can be plugged with k3s, the [install](https://github.com/mudler/kubecfctl/blob/master/install) act as a wrapper to the k3s installer and runs Kubecfctl on top.

For example:

```bash
curl -L "https://raw.githubusercontent.com/mudler/kubecfctl/master/install" | sudo INTERNAL_INSTALL_KUBECFCTL_EXEC="kubecf" INSTALL_K3S_EXEC="k3s args.." bash
```
