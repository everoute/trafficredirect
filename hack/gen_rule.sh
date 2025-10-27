#!/bin/bash

VERBOSE=false

show_help() {
    cat << EOF
Usage: $0 <count> [options]
       $0 -h|--help

Generate specified number of Everoute rules and automatically apply to Kubernetes cluster.

Arguments:
  <count>    Number of rule pairs to generate (must be greater than 0)

Options:
  -h, --help  Show this help message
  -v, --verbose  Show verbose output for each rule

Examples:
  $0 5              Generate 5 rule pairs (quiet mode)
  $0 10 -v          Generate 10 rule pairs with verbose output
  $0 3 --verbose    Generate 3 rule pairs with verbose output
EOF
}

# Parse command line arguments
COUNT=""
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        *)
            if [[ -z "$COUNT" ]] && [[ "$1" =~ ^[0-9]+$ ]]; then
                COUNT=$1
                shift
            else
                echo "Error: Invalid argument: $1"
                show_help
                exit 1
            fi
            ;;
    esac
done

# Validate count parameter
if [[ -z "$COUNT" ]] || [[ "$COUNT" -lt 1 ]]; then
    echo "Error: Parameter must be a number greater than 0"
    echo "Use '$0 -h' for help"
    exit 1
fi

# Find kubectl binary
find_kubectl() {
    local found_path=$(find /var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/ -type f -executable -name kubectl 2>/dev/null | head -1)
    
    if [ -z "$found_path" ]; then
        found_path=$(find /var/lib/ -name kubectl -type f -executable 2>/dev/null | head -1)
    fi
    
    echo "$found_path"
}

KUBECTL_CMD=$(find_kubectl)

if [ -z "$KUBECTL_CMD" ]; then
    echo "Error: Cannot find kubectl binary"
    exit 1
fi

if [ ! -x "$KUBECTL_CMD" ]; then
    echo "Error: kubectl found but not executable: $KUBECTL_CMD"
    exit 1
fi

# Test if kubectl works
if ! "$KUBECTL_CMD" version --client &> /dev/null; then
    echo "Error: kubectl is not working properly"
    exit 1
fi

# Generate MAC address function
generate_mac() {
    printf '51:54:%02x:%02x:%02x:%02x\n' \
        $((RANDOM % 256)) \
        $((RANDOM % 256)) \
        $((RANDOM % 256)) \
        $((RANDOM % 256))
}

# Generate UUID function
generate_uuid() {
    if command -v uuidgen &> /dev/null; then
        uuidgen | tr -d '-' | tr '[:upper:]' '[:lower:]'
    else
        cat /dev/urandom | tr -dc 'a-f0-9' | fold -w 32 | head -1
    fi
}

# Log function with verbose check
log() {
    if [ "$VERBOSE" = true ]; then
        echo "$1"
    fi
}

# Progress indicator for non-verbose mode
show_progress() {
    if [ "$VERBOSE" = false ]; then
        local current=$1
        local total=$2
        local percent=$((current * 100 / total))
        printf "\rProgress: [%-50s] %d%% (%d/%d)" \
               "$(printf '#%.0s' $(seq 1 $((percent / 2))))" \
               "$percent" "$current" "$total"
    fi
}

if [ "$VERBOSE" = true ]; then
    echo "Starting to generate $COUNT rule pairs..."
    echo "Using kubectl from: $KUBECTL_CMD"
    echo "========================================"
else
    echo "Generating $COUNT rule pairs..."
fi

SUCCESS_COUNT=0

for ((i=1; i<=COUNT; i++)); do
    UUID=$(generate_uuid)
    MAC=$(generate_mac)
    
    # Create temporary files
    EGRESS_FILE=$(mktemp)
    INGRESS_FILE=$(mktemp)
    
    # Generate egress rule
    cat > "$EGRESS_FILE" << EOF
apiVersion: tr.everoute.io/v1alpha1
kind: Rule
metadata:
  name: ${UUID}-egress
  namespace: zj-test-tr
spec:
  direct: egress
  match:
    srcMac: ${MAC}
  option:
    towerVM: fake-vm-cmg07yu5n0x2q0858108vy1iz
EOF

    # Generate ingress rule
    cat > "$INGRESS_FILE" << EOF
apiVersion: tr.everoute.io/v1alpha1
kind: Rule
metadata:
  name: ${UUID}-ingress
  namespace: zj-test-tr
spec:
  direct: ingress
  match:
    dstMac: ${MAC}
  option:
    towerVM: fake-vm-cmg07yu5n0x2q0858108vy1iz
EOF

    # Apply rules
    EGRESS_SUCCESS=false
    INGRESS_SUCCESS=false

    log "Generating rule pair $i/$COUNT..."
    log "Applying egress rule: $UUID-egress"
    
    if "$KUBECTL_CMD" apply -f "$EGRESS_FILE" &> /dev/null; then
        EGRESS_SUCCESS=true
        log "✓ Successfully applied egress rule"
    else
        log "✗ Failed to apply egress rule"
    fi
    rm -f "$EGRESS_FILE"

    log "Applying ingress rule: $UUID-ingress"
    if "$KUBECTL_CMD" apply -f "$INGRESS_FILE" &> /dev/null; then
        INGRESS_SUCCESS=true
        log "✓ Successfully applied ingress rule"
    else
        log "✗ Failed to apply ingress rule"
    fi
    rm -f "$INGRESS_FILE"

    if $EGRESS_SUCCESS && $INGRESS_SUCCESS; then
        ((SUCCESS_COUNT++))
        log "✓ Successfully applied rule pair: $UUID, MAC: $MAC"
    else
        log "✗ Failed to apply rule pair: $UUID, MAC: $MAC"
    fi
    
    log "---"
    
    # Show progress in non-verbose mode
    if [ "$VERBOSE" = false ]; then
        show_progress $i $COUNT
    fi
done

# Complete progress line
if [ "$VERBOSE" = false ]; then
    echo
fi

echo "========================================"
echo "Completed!"
echo "Successfully generated and applied: $SUCCESS_COUNT/$COUNT rule pairs"
echo "All MAC addresses start with: 51:54"