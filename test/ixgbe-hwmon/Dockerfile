FROM --platform=linux/amd64 vyos-custom:test

WORKDIR /build

# Copy pre-extracted ixgbe driver source
COPY ethernet-linux-ixgbe-main /build/ethernet-linux-ixgbe-main

WORKDIR /build/ethernet-linux-ixgbe-main/src
