#!/bin/bash
# Remove the invalid toolchain line and set Go to 1.23
go mod edit -go=1.23 -toolchain=go1.23.4

# Clean up dependencies
go mod tidy
