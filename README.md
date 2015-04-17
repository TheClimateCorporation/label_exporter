# Prometheus label_exporter

[![Build Status](https://travis-ci.org/theClimateCorporation/label_exporter.svg)](https://travis-ci.org/theClimateCorporation/label_exporter)

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
Here are a few examples of how to fetch metris from other Prometheus
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
