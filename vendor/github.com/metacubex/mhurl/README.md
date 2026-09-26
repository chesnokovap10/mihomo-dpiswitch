# MHURL
[![Go Reference](https://pkg.go.dev/badge/github.com/metacubex/mhurl.svg)](https://pkg.go.dev/github.com/metacubex/mhurl)

parses URLs support multiple hosts.

Copy and modify from https://github.com/golang/go/blob/go1.26.1/src/net/url/url.go

The reason is that the origin `Parse()` func does not support multiple hosts in the Host part after Go 1.26.
See the original issue for more details:

* https://github.com/golang/go/issues/75859
* https://github.com/golang/go/issues/78077