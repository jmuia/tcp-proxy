#!/bin/sh

echo attempting to listen on $1

# Use tee to duplicate to stderr so we can see the
# messages in docker logs, too.
/usr/bin/ncat -k -c "/usr/bin/tee -a /dev/stderr" -l $1
