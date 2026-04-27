#!/bin/bash

kubectl proxy &
PROXY_PID=$!
sleep 2

trap "kill $PROXY_PID 2>/dev/null" EXIT

terminating_ns=$(kubectl get namespaces --field-selector status.phase=Terminating -o jsonpath='{.items[*].metadata.name}')

if [ -z "$terminating_ns" ]; then
  echo "$(date): No namespaces in Terminating state"
  exit 0
fi

for ns in $terminating_ns; do
  echo "$(date): Cleaning up namespace: $ns"
  
  # FIRST: Remove finalizers from Ray CRDs
  for resource in rayclusters.ray.io rayservices.ray.io rayjobs.ray.io; do
    for name in $(kubectl get "$resource" -n "$ns" -o jsonpath='{.items[*].metadata.name}' 2>/dev/null); do
      echo "  Removing finalizers from $resource/$name"
      kubectl patch "$resource" "$name" -n "$ns" --type=json -p='[{"op":"remove","path":"/metadata/finalizers"}]' 2>/dev/null &
      sleep 0.5
      kill $! 2>/dev/null
    done
  done
  
  # SECOND: Remove finalizers from pods
  for pod in $(kubectl get pods -n "$ns" -o jsonpath='{.items[*].metadata.name}' 2>/dev/null); do
    kubectl patch pod "$pod" -n "$ns" --type=json -p='[{"op":"remove","path":"/metadata/finalizers"}]' 2>/dev/null &
    sleep 0.2
    kill $! 2>/dev/null
  done
  
  # THIRD: Quick delete attempts
  kubectl delete pods --all -n "$ns" --force --grace-period=0 --wait=false 2>/dev/null &
  sleep 1
  kill $! 2>/dev/null
  
  kubectl delete all --all -n "$ns" --force --grace-period=0 --wait=false 2>/dev/null &
  sleep 1
  kill $! 2>/dev/null

  # Finalize the namespace
  echo "$(date): Finalizing namespace: $ns"
  kubectl get namespace "$ns" -o json 2>/dev/null | jq '.spec = {"finalizers":[]}' > /tmp/finalize-$ns.json
  curl -s -k -H "Content-Type: application/json" -X PUT --data-binary @/tmp/finalize-$ns.json "127.0.0.1:8001/api/v1/namespaces/$ns/finalize" > /dev/null
  rm -f /tmp/finalize-$ns.json
  
  echo "$(date): Done with $ns"
done

echo "$(date): Cleanup complete"