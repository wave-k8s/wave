#!/usr/bin/env bash

set -eu
set -o pipefail

helm --help > /dev/null 2>&1 || {
  echo "helm is not installed"
  exit 1
}

kubectl --help > /dev/null 2>&1 || {
  echo "kubectl is not installed"
  exit 1
}

if [[ "$1" == "clean" ]]; then
  helm uninstall wave || true
  kubectl delete configmap test test-completed  || true
  kubectl delete role test  || true
  kubectl delete rolebinding test  || true
  kubectl delete deployment test  || true
  exit 0
fi

MINIKUBE_ALREADY_RUNNING=0
kubectl get node minikube >/dev/null 2>&1 && MINIKUBE_ALREADY_RUNNING=1

minikube --help > /dev/null 2>&1 || {
  echo "minikube is not installed"
  exit 1
}

KUBERNETES_VERSION=${1:-v1.21}

[ $MINIKUBE_ALREADY_RUNNING  -eq 0 ] && {
	echo Starting minikube...
	minikube start --kubernetes-version "$KUBERNETES_VERSION"
	trap "minikube delete" EXIT
}

eval $(minikube -p minikube docker-env)
docker build -f ./Dockerfile -t wave-local:local .

echo Installing wave...
if [ "$1" = "production" ]; then
	# Install cert-manager
	kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.5/cert-manager.yaml
	while [ "$(kubectl get pods -n cert-manager | grep 'webhook' | grep -c '1/1')" -ne 1 ]; do echo Waiting for \"cert-manager-webhook\" to start; sleep 10; done
	# Production setup
	helm install wave charts/wave --set image.name=wave-local --set image.tag=local -f hack/production.yaml
else
	# Default install without values
	helm install wave charts/wave --set image.name=wave-local --set image.tag=local
fi

while [ "$(kubectl get pods -A | grep -cEv 'Running|Completed')" -gt 1 ]; do echo Waiting for \"cluster\" to start; sleep 10; done

echo Creating test resources...
kubectl create -f - <<'EOF'
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  test: init
EOF

kubectl create -f - <<'EOF'
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: test
rules:
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["patch", "create", "get"]
EOF

kubectl create -f - <<'EOF'
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: test
subjects:
- kind: ServiceAccount
  name: default
roleRef:
  kind: Role
  name: test
  apiGroup: rbac.authorization.k8s.io
EOF

kubectl create -f - <<'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
  annotations:
    wave.pusher.com/update-on-config-change: "true"
    wave.pusher.com/extra-configmaps: "test/test"
    wave.pusher.com/extra-secrets: "test/test"
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: test
        image: nixery.dev/shell/kubectl
        command:
          - /bin/sh
          - -ec
          - |
            sleep 60
            if [ $(cat /etc/config/test) = "updated" ]; then
              kubectl create configmap test-completed
            elif [ $(cat /etc/config/test) = "init" ]; then
              kubectl patch configmap test --type merge -p '{"data":{"test":"updated"}}'
            fi
            sleep infinity
        volumeMounts:
        - name: config
          mountPath: /etc/config
      volumes:
      - name: config
        configMap:
          name: test
EOF

ctr=0
while ! kubectl get cm test-completed; do
  echo Waiting for test to complete
  sleep 10
  ctr=$((ctr+1))
  if [ "$ctr" -gt 60 ]; then
	echo "Test failed"
	exit 1
  fi
done

echo Test passed
exit 0
