[![GitHub release](https://img.shields.io/github/release/sgaunet/kubetrainer.svg)](https://github.com/sgaunet/kubetrainer/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/sgaunet/kubetrainer)](https://goreportcard.com/report/github.com/sgaunet/kubetrainer)
![GitHub Downloads](https://img.shields.io/github/downloads/sgaunet/kubetrainer/total)

This project will try to guide you to get a local kubernetes environment in order to:

* learn
* try
* break
* retry
* ...


This project is tested under Linux for the moment.


* You have to install docker
* and [kind](https://kind.sigs.k8s.io/)
* and [task](https://taskfile.dev/)

Done ?

Ok, let's begin by creating the cluster.

# Create the kubernetes cluster

```bash
$ task create-cluster
task: [create-cluster] kind create cluster --config kind-config.yaml
Creating cluster "kind" ...
...
```

# Connect to the cluster

```bash
$ kubectl cluster-info --context kind-kind  # execute this command to be able to contact the kubernetes cluster
```

What should I do now ?

Follow tutorials in the [DOCS](DOCS) directory.
