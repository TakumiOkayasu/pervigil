#!/bin/bash
set -e

echo "=== ixgbe HWMON Test ==="

# Check module exists
if [ ! -f /output/ixgbe.ko ]; then
    echo "ERROR: /output/ixgbe.ko not found"
    exit 1
fi

echo "Module: $(ls -la /output/ixgbe.ko)"
echo ""

# Get kernel version
KERNEL_VERSION=$(ls /lib/modules/ | grep -v "^$" | head -1)
echo "Kernel: ${KERNEL_VERSION}"

# Unload existing module if loaded
if lsmod | grep -q "^ixgbe"; then
    echo "Unloading existing ixgbe..."
    rmmod ixgbe || true
fi

# Load new module
echo ""
echo "=== Loading ixgbe with HWMON ==="
insmod /output/ixgbe.ko
lsmod | grep ixgbe

# Check hwmon
echo ""
echo "=== Checking hwmon ==="
if ls /sys/class/hwmon/hwmon*/name 2>/dev/null; then
    for hwmon in /sys/class/hwmon/hwmon*/; do
        name=$(cat "${hwmon}name" 2>/dev/null || echo "unknown")
        echo "hwmon: $hwmon -> $name"
        if [[ "$name" == *ixgbe* ]] || [[ "$name" == *"X540"* ]]; then
            echo "FOUND: ixgbe hwmon!"
            cat "${hwmon}temp1_input" 2>/dev/null && echo " (temp available)" || echo " (no temp)"
        fi
    done
else
    echo "No hwmon devices found"
fi

# Check network interfaces
echo ""
echo "=== Network interfaces (ixgbe) ==="
for iface in /sys/class/net/*/device/driver; do
    if readlink "$iface" 2>/dev/null | grep -q ixgbe; then
        ifname=$(echo "$iface" | cut -d/ -f5)
        echo "Interface: $ifname"
        # Check for hwmon under the interface
        if ls /sys/class/net/"$ifname"/device/hwmon/*/temp1_input 2>/dev/null; then
            temp=$(cat /sys/class/net/"$ifname"/device/hwmon/*/temp1_input 2>/dev/null)
            echo "  Temperature: $((temp / 1000))°C"
        fi
    fi
done

echo ""
echo "=== Test Complete ==="
