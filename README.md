<img src="./wave-logo.svg" width=150 height=150 alt="Wave Logo"/>

# Wave

Wave watches Deployments within a Kubernetes cluster and ensures that each
Deployment's Pods always have up to date configuration.

By monitoring ConfigMaps and Secrets mounted by a Deployment, Wave can trigger
a Rolling Update of the Deployment when the mounted configuration is changed.

Please have a look at our [documentation at github](https://github.com/wave-k8s/wave).