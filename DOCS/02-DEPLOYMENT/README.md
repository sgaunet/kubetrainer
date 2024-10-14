Now we will create a deployment in the cluster.

```bash
$ kubectl apply -f deployment.yaml
deployment.apps/http-echo created
```

A deployement is a higher level abstraction that manages pods. It allows you to define the number of replicas you want to run and how to update them.

You can see the deployment with:

```bash
$ kubectl get deployments
NAME        READY   UP-TO-DATE   AVAILABLE   AGE
http-echo   1/1     1            1           1m
```

You can also see the deployment's details with:

```bash
$ kubectl describe deployment http-echo
Name:                   http-echo
Namespace:              default
...
```

You can access the deployment by port-forwarding the deployment's port to your local machine:

```bash
$ kubectl port-forward deploy/http-echo 8080:8080
Forwarding from ...
```

Open a web browser and go to [http://localhost:8080/](http://localhost:8080/). You should see the http-echo service.

Scale the deployment to 3 replicas:

```bash
$ kubectl scale --replicas=3 deployment/http-echo
deployment.apps/http-echo scaled
```

Now, delete the deployment:

```bash
$ kubectl delete -f deployment.yaml
```
