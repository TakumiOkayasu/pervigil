#!/bin/bash
set -e

echo "=== ixgbe HWMON Build ==="

# Find kernel version
KERNEL_VERSION=$(ls /lib/modules/ | grep -v "^$" | head -1)
echo "Kernel: ${KERNEL_VERSION}"

KSRC="/lib/modules/${KERNEL_VERSION}/build"
if [ ! -d "$KSRC" ]; then
    echo "ERROR: Kernel headers not found: $KSRC"
    echo "Available in /lib/modules/:"
    ls -la /lib/modules/
    exit 1
fi

echo ""
echo "=== Building with HWMON ==="
cd /build/ethernet-linux-ixgbe-main/src
make KSRC="$KSRC" CFLAGS_EXTRA="-DIXGBE_HWMON" clean || true
make KSRC="$KSRC" CFLAGS_EXTRA="-DIXGBE_HWMON" -j$(nproc)

echo ""
echo "=== Result ==="
ls -la *.ko

# Verify HWMON
echo ""
echo "=== HWMON symbols check ==="
if strings ixgbe.ko | grep -qi hwmon; then
    echo "OK: HWMON support detected"
else
    echo "WARN: HWMON symbols not found"
fi

# Copy to output
if [ -d /output ]; then
    cp *.ko /output/
    echo ""
    echo "=== Copied to /output ==="
    ls -la /output/
fi
