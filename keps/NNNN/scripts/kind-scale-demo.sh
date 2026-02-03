#!/usr/bin/env bash
# scripts/kind-scale-demo.sh
# Simple scalability demo for local testing using kind + kubectl
# Creates a kind cluster (if not exists), deploys a namespace and then creates N simple pods
# Usage: ./scripts/kind-scale-demo.sh [num_pods]

set -euo pipefail

NUM_PODS=${1:-100}
CLUSTER_NAME="csi-env-demo"
NAMESPACE="csi-env-demo"
POD_PREFIX="scale-test"

command -v kind >/dev/null 2>&1 || { echo "kind is required. Install from https://kind.sigs.k8s.io/"; exit 1; }
command -v kubectl >/dev/null 2>&1 || { echo "kubectl is required."; exit 1; }

# Create kind cluster if not exists
if ! kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
  echo "Creating kind cluster ${CLUSTER_NAME}..."
  kind create cluster --name "${CLUSTER_NAME}" --wait 60s
else
  echo "Kind cluster ${CLUSTER_NAME} already exists"
fi

kubectl cluster-info --context "kind-${CLUSTER_NAME}"

# Create namespace
kubectl get namespace "${NAMESPACE}" >/dev/null 2>&1 || kubectl create namespace "${NAMESPACE}"

echo "Creating ${NUM_PODS} test pods in namespace ${NAMESPACE}..."
start_time=$(date +%s)

for i in $(seq 1 ${NUM_PODS}); do
  cat <<EOF | kubectl apply -n "${NAMESPACE}" -f - >/dev/null
apiVersion: v1
kind: Pod
metadata:
  name: ${POD_PREFIX}-${i}
  labels:
    app: ${POD_PREFIX}
spec:
  containers:
  - name: sleeper
    image: busybox
    command: ["/bin/sh", "-c", "sleep 3600"]
  restartPolicy: Never
EOF
  if (( i % 20 == 0 )); then echo "  Created $i pods..."; fi
done

echo "Waiting for pods to be scheduled..."
kubectl wait --for=condition=Ready --timeout=120s pod -n "${NAMESPACE}" -l app=${POD_PREFIX} || true

end_time=$(date +%s)
elapsed=$((end_time - start_time))

echo "Created ${NUM_PODS} pods in ${elapsed}s (note: scheduling may continue after script completion)."

echo "Cleanup: to delete the namespace run: kubectl delete namespace ${NAMESPACE}"

echo "To delete kind cluster: kind delete cluster --name ${CLUSTER_NAME}"
