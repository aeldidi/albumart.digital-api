Albumart.Digital API Server
===========================

The [Albumart.digital](https://albumart.digital) server is a FastCGI server
serving the API for [Albumart.Digital](https://albumart.digital). It returns
pre-rendered HTML for the web frontend to discourage abuse from 3rd-parties
using it as a free Apple Music API.

Building
--------

This makes use of the `//go:embed` directive, which was added in Go 1.16, so
you'll need at least that version to build this. Other than that however, the
server has no dependencies, and can be built by running

```
go build
```

in this directory.

Rate Limiting
-------------

Since this API is only meant to be accessed from the web frontend and not as a
general purpose "Album Art API", it's wise to use your web server of choice to
put a rate limit in front of this server. Specifically, the `web-client` will
never access this API more than 20 times every second.

Configuration
-------------

This server is configured using environment variables, or optionally a `.env`
file containing definitions for environment variables. See `.env.example` for
a sample of what can be configured in this way.

License
-------

Everything in this repo is in the public domain. See `LICENSE` for more
information.
