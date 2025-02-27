Create an ingress controller to expose the application to the outside world. The ingress controller will be used to route traffic to the correct service based on the hostname and path. The ingress controller will be created using the NGINX Ingress Controller.

```bash
$ kubectl apply -f ingress.yaml
ingress.networking.k8s.io/http-echo-ingress created
```

You can see the ingress with:

```bash
$ kubectl get ingress
NAME               CLASS    HOSTS   ADDRESS   PORTS   AGE
http-echo-ingress   <none>   *                 80      1m
```

You can also see the ingress's details with:

```bash
$ kubectl describe ingress http-echo
Name:             http-echo-ingress
...
```

[Now you can access the application.](http://localhost:80)
