#!/bin/bash
# patch-deployment.sh
# Dynamically generates coverage infrastructure from base files at any revision
# This script generates 4 files needed for coverage instrumentation:
#   1. Dockerfile.coverage
#   2. config/overlays/coverage/kustomization.yaml
#   3. config/overlays/coverage/deployment-coverage.yaml
#   4. Makefile (appends coverage targets)
#
# Requires: yq (https://github.com/mikefarah/yq)
# Usage: ./patch-deployment.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Check we're in the right place
if [[ ! -f "${SCRIPT_DIR}/Makefile" ]]; then
    echo "Error: Run from ray-operator directory or ensure Makefile exists"
    exit 1
fi

cd "$SCRIPT_DIR"

# Check for yq
if ! command -v yq &> /dev/null; then
    echo "Warning: yq not found. Container name detection will use default."
    echo "Install with: brew install yq (macOS) or snap install yq (Linux)"
fi

echo "=== Generating coverage files for revision $(git rev-parse --short HEAD 2>/dev/null || echo 'unknown') ==="

#######################################
# 1. Generate Dockerfile.coverage
#######################################
generate_dockerfile_coverage() {
    echo "Generating Dockerfile.coverage..."
    
    # Extract Go version from go.mod
    GO_VERSION=$(grep '^go ' go.mod | awk '{print $2}' | cut -d. -f1,2)
    echo "  Detected Go version: ${GO_VERSION}"
    
    cat > Dockerfile.coverage << EOF
# Auto-generated coverage Dockerfile
# Generated from base files at revision $(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')
FROM golang:${GO_VERSION}-bookworm AS builder

# Use the full module path so coverage reports show correct source paths
WORKDIR /go/src/github.com/ray-project/kuberay/ray-operator

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# Cache deps before building and copying source
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY apis/ apis/
COPY controllers/ controllers/
COPY pkg/ pkg/
COPY rayjob-submitter/ rayjob-submitter/

# Build with coverage instrumentation
# The -cover flag instruments the binary to collect coverage data at runtime.
# Coverage data is written to the directory specified by GOCOVERDIR env var.
USER root
RUN CGO_ENABLED=1 GOOS=linux go build -cover -covermode=atomic -a -o manager main.go

# Use a base image that has a writable filesystem for coverage data
# Note: We use debian-slim instead of distroless to allow writing coverage files
FROM debian:bookworm-slim

# Create a non-root user
RUN groupadd -g 65532 nonroot && useradd -u 65532 -g nonroot -s /bin/false nonroot

# Create coverage directory with appropriate permissions
RUN mkdir -p /coverage && chown nonroot:nonroot /coverage

WORKDIR /
COPY --from=builder /go/src/github.com/ray-project/kuberay/ray-operator/manager .

# Create a wrapper script that:
# 1. Runs the manager
# 2. When SIGTERM is received, forwards it to manager and waits
# 3. After manager exits (coverage written), sleeps to allow data collection
RUN printf '#!/bin/bash\n\
trap "echo Coverage flush starting..." SIGTERM\n\
/manager "\$@" &\n\
MANAGER_PID=\$!\n\
trap "kill -TERM \$MANAGER_PID 2>/dev/null; wait \$MANAGER_PID; echo Coverage written, sleeping for collection...; sleep 30" SIGTERM\n\
wait \$MANAGER_PID\n\
echo Manager exited, sleeping for coverage collection...\n\
sleep 30' > /entrypoint.sh && chmod +x /entrypoint.sh

# Set the coverage directory - can be overridden at runtime
ENV GOCOVERDIR=/coverage

USER 65532:65532

ENTRYPOINT ["/entrypoint.sh"]
EOF

    echo "  Created Dockerfile.coverage"
}

#######################################
# 2. Generate coverage kustomize overlay
#######################################
generate_kustomize_overlay() {
    echo "Generating config/overlays/coverage/..."
    
    mkdir -p config/overlays/coverage
    
    # Find the base kustomization to reference
    # Prefer test-overrides if it exists, otherwise use default
    if [[ -d "config/overlays/test-overrides" ]]; then
        BASE_REF="../test-overrides"
        echo "  Using test-overrides as base"
    else
        BASE_REF="../../default"
        echo "  Using default as base (test-overrides not found)"
    fi
    
    # Generate kustomization.yaml
    cat > config/overlays/coverage/kustomization.yaml << EOF
# Auto-generated coverage overlay for revision $(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ${BASE_REF}

patches:
  - path: deployment-coverage.yaml
    target:
      kind: Deployment
      name: kuberay-operator
EOF

    # Detect the container name from existing deployment
    CONTAINER_NAME="kuberay-operator"
    if [[ -f "config/manager/manager.yaml" ]] && command -v yq &> /dev/null; then
        DETECTED_NAME=$(yq '.spec.template.spec.containers[0].name' config/manager/manager.yaml 2>/dev/null || echo "")
        if [[ -n "$DETECTED_NAME" && "$DETECTED_NAME" != "null" ]]; then
            CONTAINER_NAME="$DETECTED_NAME"
        fi
    fi
    echo "  Container name: ${CONTAINER_NAME}"
    
    # Generate deployment patch
    cat > config/overlays/coverage/deployment-coverage.yaml << EOF
# Auto-generated coverage deployment patch
# This patch:
# - Adds GOCOVERDIR environment variable
# - Mounts a hostPath volume for coverage data (persists across pod restarts)
# - Uses initContainer to fix permissions on hostPath
# - Removes readOnlyRootFilesystem constraint (needed to write coverage)
# - Relaxes pod security context to allow initContainer to run as root
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kuberay-operator
spec:
  template:
    spec:
      # Override pod-level security to allow initContainer to run as root
      securityContext:
        runAsNonRoot: false
      # Give time for coverage to be written on shutdown
      terminationGracePeriodSeconds: 30
      # InitContainer to fix permissions on hostPath volume
      initContainers:
      - name: fix-permissions
        image: busybox:latest
        command: ["sh", "-c", "chmod 777 /coverage"]
        volumeMounts:
        - name: coverage-data
          mountPath: /coverage
        securityContext:
          runAsUser: 0
          runAsNonRoot: false
      containers:
      - name: ${CONTAINER_NAME}
        env:
        - name: GOCOVERDIR
          value: /coverage
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
          # Must be false to allow writing coverage data
          readOnlyRootFilesystem: false
          seccompProfile:
            type: RuntimeDefault
        volumeMounts:
        - name: coverage-data
          mountPath: /coverage
      volumes:
      - name: coverage-data
        hostPath:
          path: /tmp/kuberay-coverage
          type: DirectoryOrCreate
EOF

    echo "  Created config/overlays/coverage/"
}

#######################################
# 3. Patch Makefile with coverage targets (idempotent)
#######################################
patch_makefile() {
    echo "Patching Makefile..."
    
    # Check if coverage targets already exist
    if grep -q "^docker-build-coverage:" Makefile; then
        echo "  Coverage targets already present, skipping"
        return 0
    fi
    
    # Append coverage targets to Makefile
    cat >> Makefile << 'EOF'

##@ Coverage (auto-generated by patch-deployment.sh)

docker-build-coverage: ## Build coverage-instrumented operator image.
	${ENGINE} build -t ${IMG} -f Dockerfile.coverage .

deploy-coverage: manifests kustomize ## Deploy coverage-instrumented controller to K8s cluster.
	cd config/default && $(KUSTOMIZE) edit set image kuberay/operator=${IMG}
	$(KUSTOMIZE) build config/overlays/coverage | kubectl apply --server-side=true -f -

undeploy-coverage: ## Undeploy coverage-instrumented controller.
	$(KUSTOMIZE) build config/overlays/coverage | kubectl delete -f -
EOF

    echo "  Added coverage targets to Makefile"
}

#######################################
# Main
#######################################
generate_dockerfile_coverage
generate_kustomize_overlay
patch_makefile

echo ""
echo "=== Coverage files generated successfully ==="
echo "Files created/modified:"
echo "  - Dockerfile.coverage"
echo "  - config/overlays/coverage/kustomization.yaml"
echo "  - config/overlays/coverage/deployment-coverage.yaml"
echo "  - Makefile (coverage targets appended)"
echo ""
echo "Next steps:"
echo "  1. Build coverage image:  IMG=kuberay/operator:coverage make docker-build-coverage"
echo "  2. Load into kind:        kind load docker-image kuberay/operator:coverage"
echo "  3. Deploy:                IMG=kuberay/operator:coverage make deploy-coverage"
