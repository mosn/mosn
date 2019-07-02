# sofaload

sofaload is benchmarking tool for SOFARPC. It is extended from [h2load](https://nghttp2.org/documentation/h2load-howto.html).

# Features

- P50, P75, P90, P99 latency
- Support SOFARPC streaming protocol
- Support HTTP/1.1 and HTTP2

# Build

    autoreconf -i
    automake
    autoconf
    ./configure --enable-app
    make
    sudo make install

# Usage

- basic mode.

    perform sofarpc benchmark using total 1000 requests, 4 concurrent clients, 2 max concurrent streams and 4 threads:

        sofaload -n 1000 -c 4 -m 2 -t 4 -p sofarpc sofarpc://[ip]:[port]

- timing-based mode

    perform sofarpc benchmark for 10 seconds after 5 seconds warming up period:

        sofaload -D 10 --warm-up-time=5 -c 4 -t 4 -p sofarpc sofarpc://[ip]:[port]

- qps mode

    perform sofarpc benchmark for 10 seconds with a fixed 2000 qps:

        sofaload -D 10 --qps=2000 -t 4 -p sofarpc sofarpc://[ip]:[port]

# Command Line Options

    -n, --requests=<N>  Number of requests across all clients.

    -c, --clients=<N>   Number of concurrent clients.
                        Default: 1

    -m, --max-concurrent-streams=<N>
                        Max concurrent streams to issue per session.
                        Default: 1

    -t, --threads=<N>   Number of native threads.
                        Default: 1

    -p, --no-tls-proto=<PROTOID>
                        Specify the protocol to be used.
                        Available protocols: h2c and http/1.1 and sofarpc

    -D, --duration=<N>  Specifies the main duration for the measurements
                        in case of timing-based and qps mode.

    --warm-up-time=<DURATION>
                        Specifies the time period before starting the actual
                        measurements, in case of timing-based and qps benchmarking.

    --qps=<N>           Specifies the qps for benchmarking.