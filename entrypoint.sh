#!/bin/bash
# entrypoint.sh

# Run the iota-zmq-server in the background
exec /iota-zmq-bridge

# Execute the CMD (in this case, /bin/bash)
#exec "$@"
