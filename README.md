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

```
task create-cluster
```

