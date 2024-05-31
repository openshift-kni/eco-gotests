This is a WSMAN library for Go.

**Fork of [github.com/ammmze/wsman](https://github.com/ammmze/wsman)**

It mostly adheres to the DMTF specifications at
http://www.dmtf.org/standards/wsman, except where it does not.

Right now, it can only communicate with WSMAN endpoints over HTTP/HTTPS
using Basic auth.

It has no unit tests because I don't feel like writing a WSMAN endpoint
in Go, but the SOAP and xml libraries it is based on do.
