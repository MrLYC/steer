#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Functional test for Steer operator using helm/examples hello-world chart.

Chart source:
  helm repo add examples https://helm.github.io/examples
  helm install ahoy examples/hello-world

This script:
  1) Creates a HelmRelease CR in the operator namespace
  2) Deploys the chart into a target namespace via spec.deployment.namespace (-n behavior)
  3) Ensures the target namespace is created (if enabled) and labeled steer.io/managed-by=steer
  4) Verifies Helm release secret exists in the target namespace

Usage:
  scripts/ft-helm-examples-hello-world.sh \
    --operator-ns steer-system \
    --target-ns steer-ft-hello \
    --release steer-hello

Options:
  --operator-ns   Namespace where the operator runs and where HelmRelease CR lives (required)
  --target-ns     Namespace where the chart is deployed (required)
  --release       Helm release name / HelmRelease metadata.name (required)
  --timeout       Timeout seconds for install (default: 300)
  --cleanup       Also delete the HelmRelease at end (default: false)

Requirements:
  - kubectl configured to a cluster where steer-operator is installed
  - CRDs installed: helmreleases.steer.io
EOF
}

OPERATOR_NS=""
TARGET_NS=""
RELEASE_NAME=""
TIMEOUT_SECONDS=300
CLEANUP=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --operator-ns) OPERATOR_NS="$2"; shift 2;;
    --target-ns) TARGET_NS="$2"; shift 2;;
    --release) RELEASE_NAME="$2"; shift 2;;
    --timeout) TIMEOUT_SECONDS="$2"; shift 2;;
    --cleanup) CLEANUP=true; shift 1;;
    -h|--help) usage; exit 0;;
    *) echo "Unknown arg: $1" >&2; usage; exit 2;;
  esac
done

if [[ -z "$OPERATOR_NS" || -z "$TARGET_NS" || -z "$RELEASE_NAME" ]]; then
  usage
  exit 2
fi

need() {
  command -v "$1" >/dev/null 2>&1 || { echo "Missing required command: $1" >&2; exit 1; }
}

need kubectl

echo "[1/6] Preflight"
# Some kubectl builds don't support `version --short`.
kubectl version >/dev/null
kubectl get crd helmreleases.steer.io >/dev/null
kubectl get namespace "$OPERATOR_NS" >/dev/null

TMP_DIR="$(mktemp -d)"
cleanup_tmp() { rm -rf "$TMP_DIR"; }
trap cleanup_tmp EXIT

HR_FILE="$TMP_DIR/helmrelease.yaml"
cat >"$HR_FILE" <<EOF
apiVersion: steer.io/v1alpha1
kind: HelmRelease
metadata:
  name: ${RELEASE_NAME}
  namespace: ${OPERATOR_NS}
spec:
  chart:
    source: repository
    repository:
      name: hello-world
      url: https://helm.github.io/examples
  deployment:
    namespace: ${TARGET_NS}
    createNamespace: true
    timeout: 5m
  values:
    inline: |
      # values are YAML
      # Use 0 replicas to avoid depending on image pulls in restricted networks.
      # This keeps helm --wait fast and still exercises Helm storage + resource creation.
      replicaCount: 0
EOF

echo "[2/6] Apply HelmRelease ${OPERATOR_NS}/${RELEASE_NAME} (deploy -> ${TARGET_NS})"
kubectl apply -f "$HR_FILE" >/dev/null

echo "[3/6] Wait for HelmRelease status.phase=Installed (timeout ${TIMEOUT_SECONDS}s)"
deadline=$(( $(date +%s) + TIMEOUT_SECONDS ))
while true; do
  phase="$(kubectl -n "$OPERATOR_NS" get helmrelease.steer.io "$RELEASE_NAME" -o jsonpath='{.status.phase}' 2>/dev/null || true)"
  msg="$(kubectl -n "$OPERATOR_NS" get helmrelease.steer.io "$RELEASE_NAME" -o jsonpath='{.status.message}' 2>/dev/null || true)"

  if [[ "$phase" == "Installed" ]]; then
    echo "  phase=Installed"
    break
  fi
  if [[ "$phase" == "Failed" ]]; then
    echo "HelmRelease failed: ${msg}" >&2
    exit 1
  fi
  now=$(date +%s)
  if (( now >= deadline )); then
    echo "Timed out waiting for HelmRelease to become Installed. phase=${phase} message=${msg}" >&2
    kubectl -n "$OPERATOR_NS" get helmrelease.steer.io "$RELEASE_NAME" -o yaml >&2 || true
    exit 1
  fi
  sleep 2
done

echo "[4/6] Verify target namespace exists + labeled steer.io/managed-by=steer"
kubectl get namespace "$TARGET_NS" >/dev/null
label_val="$(kubectl get namespace "$TARGET_NS" -o jsonpath='{.metadata.labels.steer\.io/managed-by}')"
if [[ "$label_val" != "steer" ]]; then
  echo "Expected namespace label steer.io/managed-by=steer, got: '${label_val}'" >&2
  kubectl get namespace "$TARGET_NS" -o yaml >&2
  exit 1
fi

echo "[5/6] Verify Helm release secret exists in target namespace"
# Helm v3 stores release info as Secrets with owner=helm and name=<release>
if ! kubectl -n "$TARGET_NS" get secret -l "owner=helm,name=${RELEASE_NAME}" >/dev/null 2>&1; then
  echo "Expected helm release secret with labels owner=helm,name=${RELEASE_NAME} in ${TARGET_NS}" >&2
  kubectl -n "$TARGET_NS" get secret -o name >&2 || true
  exit 1
fi

echo "[6/6] PASS"

if [[ "$CLEANUP" == "true" ]]; then
  echo "Cleanup: deleting HelmRelease ${OPERATOR_NS}/${RELEASE_NAME}"
  kubectl -n "$OPERATOR_NS" delete helmrelease.steer.io "$RELEASE_NAME" --ignore-not-found >/dev/null
fi
