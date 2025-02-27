Let's create a pod.

```bash
$ kubectl apply -f 01-POD/pod.yaml
pod/nginx created
```

You should see it running with:

```bash
$ kubectl get pods
NAME    READY   STATUS    RESTARTS   AGE
nginx   1/1     Running   0          2m
```

You can also see the pod's details with:

```bash
$ kubectl describe pod nginx
Name:         nginx
Namespace:    default
...
```

Ok, I've launching a pod, but what is a pod ?

A pod is the smallest deployable unit in Kubernetes. A pod is a group of one or more containers, with shared storage/network, and a specification for how to run the containers.

In this case, we have a pod with a single container running an nginx web server.

You can access the pod by port-forwarding the pod's port to your local machine:

```bash
$ kubectl port-forward pod/nginx 8080:80
Forwarding from
```

Let the port-forward running and open a new terminal to access the pod:

```bash
$ curl localhost:8080
<!DOCTYPE html>
<html>
<head>
<title>Welcome to nginx!</title>
...
```

You should see the default nginx welcome page.

Now, delete the pod:

```bash
kubectl delete -f pod.yaml
```