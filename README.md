# Prometheus label_exporter

[![Build Status](https://travis-ci.org/TheClimateCorporation/label_exporter.svg)](https://travis-ci.org/TheClimateCorporation/label_exporter)

Prometheus exporter to inject/override arbitrary labels on behalf of
any other exporter.

## Building and running

	make
	./label_exporter <flags>

## Running tests

	make test

## Usage

Once you have `label_exporter` running you can find it at
[localhost:9900](http://localhost:9900) (unless a different port is
configured via `-web.listen-address=":1234"`

## Metrics for the service itself

You can find the bare metrics for the service itself at:
[localhost:9900/metrics](http://localhost:9900/metrics). If you want
to have the proxy inject labels into itself, you'd use
[localhost:9900/9900/metrics](http://localhost:9900/9900/metrics)

## Metrics for another service

By default `label_exporter` **only** proxies to things on localhost.
Here are a few examples of how to fetch metrics from other Prometheus
services:

- Prometheus itself: [localhost:9900/9090/metrics](http://localhost:9900/9090/metrics)
- node_exporter: [localhost:9900/9100/metrics](http://localhost:9900/9100/metrics)

Or your own custom application listening on `8080` at
[localhost:9900/8080/metrics](http://localhost:9900/8080/metrics)

## Current list of options

```
$ ./label_exporter -h
Usage of ./label_exporter:
  -labels-dir="/tmp/target": Directory to find *.label in
  -proxy-host="localhost": Host to proxy requests against
  -web.listen-address=":9900": Address to listen on
```

## How to inject labels

The exporter consumes from two places, in this order:

1. Text files who's names are suffixed by `.label` in the `-labels-dir`.
2. Query string arguments.

First start up the exporter and see what we get without doing
anything:

`./label_exporter -labels-dir="/tmp"`

We can use `curl` to get a sense for things:

```
$ curl -s 'http://localhost:9900/9900/metrics' | head -n3
# HELP http_request_duration_microseconds The HTTP request latencies in microseconds.
# TYPE http_request_duration_microseconds summary
http_request_duration_microseconds{handler="prometheus",quantile="0.5"} 1543.706
```

Here we have unblemished Prometheus `label_exporter` metrics.

### Using plain text files

Let's pretend we're running our Prometheus stack in ec2, and we would
*actually* prefer to use amazon instance id(s) for the instance name,
versus fqdn or whatever happens to be in use. We could enable this by
running this shell command:

`echo "i-922370g2" > /tmp/instance.label`

Now let's see what our metrics look like:

```
$ curl -s 'http://localhost:9900/9900/metrics' | head -n3
# HELP http_request_duration_microseconds The HTTP request latencies in microseconds.
# TYPE http_request_duration_microseconds summary
http_request_duration_microseconds{handler="prometheus",instance="i-922370g2",quantile="0.5"} 1593.578
```

## Using query string arguments

If you happen to be using [srv records](http://prometheus.io/docs/operating/configuration/#configuration)
for service discovery, you loose the ability to use custom labels. We
can add this functionality back in... by using query string arguments
in the `/metrics_path` stanza. First let's demonstrate via curl by
adding `region=us-west` as a query string argument:

```
$ curl -s 'http://localhost:9900/9900/metrics?region=us-west' | head -n3
# HELP http_request_duration_microseconds The HTTP request latencies in microseconds.
# TYPE http_request_duration_microseconds summary
http_request_duration_microseconds{handler="prometheus",instance="i-922370g2",quantile="0.5",region="us-west"} 1512.28
```

## Use both file and query string with service discovery

In reality `region` is only known by the instances themselves, so
using the text file interface is better, but something like `service`
might be something Prometheus knows about, and can add to each job
that runs the service. Here's a contrived example:

```
# Service foobar: node_exporter
job {
  name: "foobar-node"
  sd_name: "telemetry.node.prod.api.srv.my-domain.org"
  metrics_path: "/9100/metrics?service=foobar"
}

# Service foobar: app (ie: jetty, golang, rails, etc)
job {
  name: "foobar-app"
  sd_name: "telemetry.app.prod.api.srv.my-domain.org"
  metrics_path: "/8080/metrics?service=foobar"
}
```

With the above example, we'd be using srv records like this:

```
host -t SRV telemetry.node.prod.api.srv.my-domain.org
telemetry.node.prod.api.srv.my-domain.org has SRV record 0 0 9900 ec2-1-2-3-4.compute-1.amazonaws.com.

host -t SRV telemetry.app.prod.api.srv.my-domain.org
telemetry.app.prod.api.srv.my-domain.org has SRV record 0 0 9900 ec2-1-2-3-4.compute-1.amazonaws.com.
```

When Prometheus scrapes the instances, it'll wind up hitting the
following urls

1. node_exporter: `ec2-1-2-3-4.compute-1.amazonaws.com:9900/9100/metrics?service=foobar`
2. jetty/golang/python: `ec2-1-2-3-4.compute-1.amazonaws.com:9900/8080/metrics?service=foobar`

This would enable queries to Promethus to reference
`{service="foobar"}` for both system and application metrics, and
without either `node_exporter` or the application itself... having to
say so.

## How is this different than using `/metrics`?

All of the things being done here could be managed by adding custom
labels to each `/metrics` endpoint. The `label_exporter` is an
optional mechanism that makes it a bit easier *maybe* :)
