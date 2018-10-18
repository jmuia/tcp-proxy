# tcp-proxy

A TCP middle proxy and load balancer.

--

Developed as an exercise after reading [Introduction to modern network load balancing and proxying](https://blog.envoyproxy.io/introduction-to-modern-network-load-balancing-and-proxying-a57f6ff80236).

A trial by fire in learning Go idioms and concurrency patterns.

## Features
- Concurrent request handling via goroutines.
- Active TCP health checking.
- Load balancing to _healthy_ backends (random or [power of 2 choices](https://brooker.co.za/blog/2012/01/17/two-random.html)).
- Metrics collection/reporting (requests, errors, tx/rx, health -- so far).
- Poor man's graceful shutdown.
- Service discovery via static configuration.

## Non-Features
- Direct server return.
- Robust connection tracking.
- Consistent hashing fallback.
- TLS termination.
- SNI-based routing.

## Usage
```
Usage: ./tcp-proxy [OPTIONS] <BACKEND>...
  -laddr string
    	address to listen on (default ":4000")
  -lb value
    	load balancer algorithm (RANDOM|P2C) (default P2C)
  -timeout duration
    	backend dial timeout (default 3s)

Example:
  ./tcp-proxy \
	-laddr localhost:4000 \
	-timeout 3s \
	-lb random \
	localhost:8001 \
	localhost:8002
```

## Docker
The easiest way to see it in action is with Docker.

The project has both a Dockerfile for the proxy as well an an `example/`
directory that includes all that's needed to spin up a test cluster.

### docker-compose

```
# docker-compose version 1.22.0, build f46880f

$ cd example/

# Build it.
$ docker-compose build

# Start up the proxy and 3 ncat echo servers.
$ docker-compose up -d

# Follow the logs in a separate terminal.
$ docker-compose logs -f

# Send some data; check the backend that
# served the connection in the logs.
$ nc localhost 4000

# Try using seq and xargs to spam some messages (OS X syntax).
$ seq 1 | xargs -I{} -L1 -P 10 sh -c 'echo hi {}! | nc -i1 localhost 4000'

# Kill a backend and see what happens when you send messages.
$ docker-compose stop service2

# Does it change after you see the log indicating it's now UNHEALTHY?

# Bring it back up and repeat.
$ docker-compose start service2
```
