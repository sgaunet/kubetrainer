
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
