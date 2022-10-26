Albumart.Digital API Server
===========================

The [Albumart.digital](https://albumart.digital) server is an HTTP server
serving the API for [Albumart.Digital](https://albumart.digital).

Building
--------

The server has no dependencies, and can be built by running

```
go build
```

in this directory.

Configuration
-------------

This server is configured using environment variables, or optionally a `.env`
file containing definitions for environment variables. See `.env.example` for
a sample of what can be configured in this way.

At runtime, the log level can be configured by hitting the config endpoint with
a GET request, setting the query `loglevel` to whatever you want. Setting
`loglevel` to DEBUG outputs a lot of intermediate information which can be
helpful when diagnosing and fixing bugs.

License
-------

The code is licensed as 0BSD, meaning everything in this repo can be used for
any purpose without needing to give credit. See `LICENSE` for more information.
