# Configuration

A configuration file is *required* for Sake but environment variables can be used to override configuration. A configuration file may be specified with the `-c` or `--config` switches or with the `SAKE_CONFIG_PATH` environment variable.

An example configuration file is included at `/config.example.toml`.

```toml
# storage driver to use for transaction persistence (env: SAKE_STORAGE)
storage = ""

[log]
# minimum event level to log (env: SAKE_LOG_LEVEL)
level = "info" # `error`, `warn`, `info`, or `debug`

# log output format (env: SAKE_LOG_FORMATTER)
formatter = "text" # `text` or `json`


[http]
# address for the http server to listen on (env: SAKE_HTTP_ADDR)
addr = ":8889" # :port

```
