Please apply again the deployment manifest:

```bash
$ kubectl apply -f ../02-DEPLOYMENT/deployment.yaml
deployment.apps/http-echo created
```

Now we will create a service in the cluster.

```bash
$ kubectl apply -f service.yaml
service/http-echo created
```

What's a service ? A service is an abstraction that defines a logical set of pods and a policy by which to access them. Services enable a loose coupling between dependent pods.

Like with pod and deployment, you can see the service with:

```bash
$ kubectl get services
NAME        TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
http-echo   ClusterIP...
```

Like with pod and deployment, you can also see the service's details with:

```bash
$ kubectl describe service http-echo
Name:              http-echo
Namespace:         default
...
```

You can access the service by port-forwarding the service's port to your local machine:

```bash
$ kubectl port-forward service/http-echo  "8080:8080"
Forwarding from...
```

Open a web browser and go to [http://localhost:8080/](http://localhost:8080/). You should see the http-echo service.
