# simple-http-server
 Simple implementation of an HTTP server

HTTP server that processes GET requests for available files.
Implements `OK` and `Not Found` responses, with concurrency.

To build:

```
    go build httpServer.go
```

To run (`httpServer.exe` on Windows):

```
    httpServer <port number>
```

For example, `httpServer 8080`.

Use your browser to go to `127.0.0.1:<port number>`. The files available in the directory will be served upon being requested.
