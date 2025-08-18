#!/bin/bash

# Ensure the current directory is the workspace root
cd /go/src/app

# Execute the command passed to the entrypoint
exec "$@"
