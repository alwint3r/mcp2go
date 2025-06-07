#!/bin/zsh

# Exit on error
set -e

# Ensure we're in the project root directory
if [[ ! -f "go.mod" ]]; then
    echo "Error: This script must be run from the project root directory"
    exit 1
fi

# Create build directory if it doesn't exist
mkdir -p build

# Find all directories in cmd/ and build each one
for cmd_dir in examples/*/; do
    if [ -d "${cmd_dir}" ]; then
        # Get the name of the command (directory name)
        cmd_name=$(basename "${cmd_dir}")
        
        echo "Building ${cmd_name}..."
        
        # Build the binary and place it in the build directory
        go build -o "build/${cmd_name}" "./examples/${cmd_name}"
        
        echo "✓ Built ${cmd_name}"
    fi
done

echo "Build complete! Binaries are in the build directory"
