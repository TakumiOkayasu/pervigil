#!/bin/bash
set -e

echo "=== ixgbe HWMON Build Test ==="
echo "Kernel: $(uname -r)"
echo ""

# Find kernel headers
KERNEL_VERSION=$(ls /lib/modules/ | head -1)
echo "Using kernel headers: ${KERNEL_VERSION}"

# Check if headers exist
if [ ! -d "/lib/modules/${KERNEL_VERSION}/build" ]; then
    echo "ERROR: Kernel headers not found at /lib/modules/${KERNEL_VERSION}/build"
    echo "Available modules:"
    ls -la /lib/modules/
    exit 1
fi

echo ""
echo "=== Building ixgbe with HWMON support ==="

# Build with HWMON enabled
make KSRC="/lib/modules/${KERNEL_VERSION}/build" CFLAGS_EXTRA="-DIXGBE_HWMON" 2>&1

if [ $? -eq 0 ]; then
    echo ""
    echo "=== Build SUCCESS ==="
    ls -la *.ko

    # Check if HWMON symbols are present
    echo ""
    echo "=== Checking HWMON symbols ==="
    if nm ixgbe.ko 2>/dev/null | grep -i hwmon; then
        echo "HWMON symbols found!"
    else
        echo "WARNING: No HWMON symbols found in module"
    fi

    # Show module info
    echo ""
    echo "=== Module info ==="
    modinfo ./ixgbe.ko | head -20
else
    echo ""
    echo "=== Build FAILED ==="
    exit 1
fi
