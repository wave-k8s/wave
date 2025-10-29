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

MINIKUBE_ALREADY_RUNNING=0
kubectl get node minikube >/dev/null 2>&1 && MINIKUBE_ALREADY_RUNNING=1

minikube --help > /dev/null 2>&1 || {
  echo "minikube is not installed"
  exit 1
}

KUBERNETES_VERSION=${1:-v1.21}
THROTTLE_INTERVAL=${2:-10s}

[ $MINIKUBE_ALREADY_RUNNING  -eq 0 ] && {
	echo Starting minikube...
	minikube start --kubernetes-version "$KUBERNETES_VERSION"
	trap "minikube delete" EXIT
}

eval $(minikube -p minikube docker-env)
docker build -f ./Dockerfile -t wave-local:local .

echo Installing wave with minUpdateInterval=${THROTTLE_INTERVAL}...
helm install wave charts/wave --set image.name=wave-local --set image.tag=local --set minUpdateInterval=${THROTTLE_INTERVAL}

while [ "$(kubectl get pods -n default | grep -cE 'wave-wave')" -gt 1 ]; do echo Waiting for \"wave\" to be scheduled; sleep 10; done

while [ "$(kubectl get pods -A | grep -cEv 'Running|Completed')" -gt 1 ]; do echo Waiting for \"cluster\" to start; sleep 10; done

echo Creating test resources...
kubectl apply -f - <<'EOF'
apiVersion: v1
kind: ConfigMap
metadata:
  name: throttle-test
data:
  counter: "0"
  timestamp: "0"
EOF

kubectl apply -f - <<'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: throttle-test
  annotations:
    wave.pusher.com/update-on-config-change: "true"
spec:
  replicas: 1
  selector:
    matchLabels:
      app: throttle-test
  template:
    metadata:
      labels:
        app: throttle-test
    spec:
      containers:
      - name: test
        image: nixery.dev/shell/bash
        command:
          - /bin/bash
          - -c
          - |
            echo "Pod started at $(date +%s)"
            echo "Counter: $(cat /etc/config/counter)"
            sleep infinity
        volumeMounts:
        - name: config
          mountPath: /etc/config
      volumes:
      - name: config
        configMap:
          name: throttle-test
EOF

echo Waiting for initial deployment to be ready...
kubectl wait --for=condition=available --timeout=60s deployment/throttle-test

# Get the initial pod name and creation time
INITIAL_POD=$(kubectl get pods -l app=throttle-test -o jsonpath='{.items[0].metadata.name}')
INITIAL_CREATION=$(kubectl get pod $INITIAL_POD -o jsonpath='{.metadata.creationTimestamp}')
echo "Initial pod: $INITIAL_POD created at $INITIAL_CREATION"

# Record start time
START_TIME=$(date +%s)

echo ""
echo "Testing update throttling by rapidly updating the ConfigMap..."
echo "Expected behavior: Updates should be throttled to minimum interval of ${THROTTLE_INTERVAL}"
echo ""

# Rapidly update the ConfigMap 10 times
for i in 1 2 3 4 5 6 7 8 9 0; do
  echo "Update $i at $(date +%H:%M:%S)"
  kubectl patch configmap throttle-test --type merge -p "{\"data\":{\"counter\":\"$i\",\"timestamp\":\"$(date +%s)\"}}"
  sleep 1
done

echo ""
echo "Waiting 15 seconds to observe throttling behavior..."
sleep 15

# Check how many pod restarts/updates occurred
echo ""
echo "Checking deployment update history..."
kubectl get replicasets -l app=throttle-test -o wide

# Get all pods that were created (including terminated ones)
echo ""
echo "Checking pod creation times..."
POD_COUNT=$(kubectl get pods -l app=throttle-test --show-all 2>/dev/null | grep -c throttle-test || kubectl get pods -l app=throttle-test | grep -c throttle-test)
echo "Total pods created: $POD_COUNT"

# Get the deployment's pod template hash changes
HASH_CHANGES=$(kubectl get replicasets -l app=throttle-test -o jsonpath='{range .items[*]}{.metadata.creationTimestamp}{"\t"}{.spec.template.spec.containers[0].image}{"\t"}{.spec.replicas}{"\n"}{end}')
echo ""
echo "ReplicaSet history (creation time, image, replicas):"
echo "$HASH_CHANGES"

# Count how many distinct replicasets were created
RS_COUNT=$(kubectl get replicasets -l app=throttle-test --no-headers | wc -l)
echo ""
echo "Number of ReplicaSets created: $RS_COUNT"

# Check wave controller logs for throttling messages
echo ""
echo "Wave controller logs (throttling messages):"
kubectl logs -l app=wave --tail=50 | grep -i "throttl\|delayed" || echo "No throttling messages found in logs"

# Verify throttling worked
echo ""
echo "=== Test Results ==="
if [ "$RS_COUNT" -le 6 ]; then
  echo "✓ PASS: Throttling is working correctly"
  echo "  - ConfigMap was updated 10 times rapidly"
  echo "  - Only $RS_COUNT deployment updates occurred (expected ≤6 with ${THROTTLE_INTERVAL} throttling)"
  exit 0
else
  echo "✗ FAIL: Throttling may not be working correctly"
  echo "  - ConfigMap was updated 10 times rapidly"
  echo "  - $RS_COUNT deployment updates occurred (expected ≤6 with ${THROTTLE_INTERVAL} throttling)"
  exit 1
fi
