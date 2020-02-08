# Example of the standard "common" block

```toml
[common]

# Applications name
name = "My service"

# Applications description
description = "The simple sample service"

# Applications class
class = "relay"

# Write logs in local time, by default in UTC
#log-local-time = true

# Directory where the logs are written. If a relative path is specified, then it is used relative to the executable file directory
log-dir = "logs"

# Logging level
# Valid valued EMERG, ALERT, CRIT, ERR, WARNING, NOTICE, INFO, DEBUG, TRACE1, TRACE2, TRACE3, TRACE4
# Default: DEBUG
log-level = "INFO"

# Log buffer size in bytes, data is pushed to the file when buffer is full, 0 - do not use buffering
log-buffer-size = 512

# If log buffering is used (log-buffer-size> 0), then this is the time in seconds after which automatic pushing to the file of a buffer that is not completely filled is performed.
# By default (value of 0) 1 second used
log-buffer-delay = 3

# Maximum line length in the log
log-max-string-len = 10000

# The maximum number of cores that the application can use. Default 0 - all available
go-max-procs = 4

# The period to write to the memory statistics statistics log in seconds
mem-stats-period = 1800

# Logging level for memory usage statistics
mem-stats-level = "INFO"

# Statistics collection period in seconds
load-avg-period = 60

# Is profiler enabled by default?
profiler-enabled = false 

# Extended profiling. Do not turn on without need!
deep-profiling = false

# Disabled endpoints
disabled-endpoints = []

# The minimum data size for gzip packing if it required when sending via HTTP. Smaller size will be send without packing
# 0  - do not pack 
# <0 - always pack
min-size-for-gzip = 256
```