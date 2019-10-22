<h3 align="center">imgur-dump</h3>

Go crawler for the upcoming Imgur dump.

```
Usage of ./imgur-dump:
  -expvar-bind string
        Where to run expvar HTTP server (off to disable) (default ":6960")
  -fasthttp
        Use fasthttp (HTTP/1.1) library instead of stdlib HTTP
  -id-format string
        ID format to scrape (id5, id7, both) (default "both")
  -id-list string
        List with downloaded IDs (default "./ids.txt")
  -out-dir string
        Directory containing images (default "./images")
  -report-interval duration
        Report interval (default 1s)
  -routines int
        Number of instances to run in parallel (default 4)
  -timeout duration
        Request timeout (default 10s)
```

expvars:
 - `reqs`: Number of requests
 - `failed`: Number of errors
 - `done`: Number of images downloaded

### Netdata setup

Enable the `python.d` plugin in `/etc/netdata/netdata.conf`
```toml
[plugins]
    python.d = yes
```

Enable the `go_expvar` plugin in `/etc/netdata/python.d.conf`
```yaml
go_expvar: yes
```

Register the custom variables:
`./netdata/go_expvar.conf >> /etc/netdata/python.d/go_expvar.conf`

Finally, restart netdata
