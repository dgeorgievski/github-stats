Custom github metrics
------------------------

GitHub API is not providing all metrics, like number of contributors and commits, about repositories. The main purpose of this app was to explore GitHub API capabiliites, HTTP requets limitations and integrations with other monitoring services like [InfluxDB Telegraf](https://www.influxdata.com/time-series-platform/telegraf/) plugin framework.

Although, it works this app is not scalable enough to handle many repositories, in which case you might want to explore data streaming and processing solutions like Spark Streaming.

This app was designed to work as external executable for a Telegraf plugin called [execd](https://github.com/influxdata/telegraf/blob/release-1.20/plugins/inputs/execd/README.md), after which output plugins could process and stream the metrics to an InfluxDB plugin.

In case the binary is executed from the command line, one empy line, `\n`, is required to be sent to the process STDOUT to initiate the metrics aggregation loop.
```shell
echo | ./github --config ./config.yaml
```

# Build and run
```shell
 go build -o github

 github --configuure
```


