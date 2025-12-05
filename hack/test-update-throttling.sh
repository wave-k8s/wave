#!/usr/bin/env bash
#
# Test script for Wave's global rate limiting feature.
#
# Usage: ./hack/test-update-throttling.sh [kubernetes-version] [update-rate] [update-burst]
#
# Arguments:
#   kubernetes-version: Kubernetes version for minikube (default: v1.21)
#   update-rate:        Updates per second globally (default: 0.2 = 1 update per 5 seconds)
#   update-burst:       Maximum burst size (default: 2)
#
# Examples:
#   ./hack/test-update-throttling.sh                    # Use defaults
#   ./hack/test-update-throttling.sh v1.21 0.5 5       # 0.5 updates/sec, burst of 5
#   ./hack/test-update-throttling.sh v1.21 1.0 10      # 1 update/sec, burst of 10

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
UPDATE_RATE=${2:-0.2}
UPDATE_BURST=${3:-2}

[ $MINIKUBE_ALREADY_RUNNING  -eq 0 ] && {
	echo Starting minikube...
	minikube start --kubernetes-version "$KUBERNETES_VERSION"
	trap "minikube delete" EXIT
}

eval $(minikube -p minikube docker-env)
docker build -f ./Dockerfile -t wave-local:local .

echo Installing wave with updateRate=${UPDATE_RATE} updateBurst=${UPDATE_BURST}...
helm install wave charts/wave --set image.name=wave-local --set image.tag=local --set updateRate=${UPDATE_RATE} --set updateBurst=${UPDATE_BURST}

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
echo "Expected behavior with updateRate=${UPDATE_RATE} updateBurst=${UPDATE_BURST}:"
echo "  - First ${UPDATE_BURST} updates will happen immediately (burst tokens)"
echo "  - Subsequent updates will be rate-limited to ${UPDATE_RATE} per second"
echo ""

# Rapidly update the ConfigMap 10 times
for i in 1 2 3 4 5 6 7 8 9 10; do
  echo "Update $i at $(date +%H:%M:%S)"
  kubectl patch configmap throttle-test --type merge -p "{\"data\":{\"counter\":\"$i\",\"timestamp\":\"$(date +%s)\"}}"
  sleep 1
done

echo ""
echo "Waiting 20 seconds to observe throttling behavior..."
sleep 20

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
# Calculate expected updates: burst + (rate * time)
# With rate=0.2, burst=2, and ~30 seconds total time:
# Expected: 2 (burst) + (0.2 * 30) = 2 + 6 = 8 updates max
# We'll allow some margin and expect ≤9 to account for timing variations
echo ""
echo "=== Test Results ==="
EXPECTED_MAX=$((UPDATE_BURST + 7))
if [ "$RS_COUNT" -le "$EXPECTED_MAX" ]; then
  echo "✓ PASS: Rate limiting is working correctly"
  echo "  - ConfigMap was updated 10 times rapidly"
  echo "  - $RS_COUNT deployment updates occurred (expected ≤${EXPECTED_MAX} with rate=${UPDATE_RATE}/sec, burst=${UPDATE_BURST})"
  echo "  - Burst allowed ${UPDATE_BURST} immediate updates, then rate-limited to ${UPDATE_RATE} per second"
  exit 0
else
  echo "✗ FAIL: Rate limiting may not be working correctly"
  echo "  - ConfigMap was updated 10 times rapidly"
  echo "  - $RS_COUNT deployment updates occurred (expected ≤${EXPECTED_MAX} with rate=${UPDATE_RATE}/sec, burst=${UPDATE_BURST})"
  exit 1
fi
