Apache SkyWalking MCP
==========

<img src="http://skywalking.apache.org/assets/logo.svg" alt="Sky Walking logo" height="90px" align="right" />

**SkyWalking-MCP**: A [Model Context Protocol][mcp] (MCP) server for integrating AI agents with Skywalking OAP and the
surrounding ecosystem.

**SkyWalking**: an APM(application performance monitor) system, especially designed for
microservices, cloud native and container-based (Docker, Kubernetes, Mesos) architectures.

## Usage

### From Source

```bash
# Clone the repository
git clone https://github.com/apache/skywalking-mcp.git
cd skywalking-mcp && go mod tidy

# Build the project
make
```

### Command-line Options

```bash
Usage:
  swmcp [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  sse         Start SSE server
  stdio       Start stdio server
  streamable  Start Streamable server

Flags:
  -h, --help               help for swmcp
      --log-command        When true, log commands to the log file
      --log-file string    Path to log file
      --log-level string   Logging level (debug, info, warn, error) (default "info")
      --read-only          Restrict the server to read-only operations
      --sw-url string      Specify the OAP URL to connect to (e.g. http://localhost:12800)
  -v, --version            version for swmcp

Use "swmcp [command] --help" for more information about a command.
```

You could start the MCP server with the following command:

```bash
# use stdio server
bin/swmcp stdio --sw-url http://localhost:12800

# or use SSE server
bin/swmcp sse --sse-address localhost:8000 --base-path /mcp --sw-url http://localhost:12800
```

### Usage with Cursor

```json
{
  "mcpServers": {
    "skywalking": {
      "command": "swmcp stdio",
      "args": [
        "--sw-url",
        "http://localhost:12800"
      ]
    }
  }
}
```

If using Docker:

`make build-image` to build the Docker image, then configure the MCP server like this:

```json
{
  "mcpServers": {
    "skywalking": {
      "command": "docker",
      "args": [
        "run",
        "--rm",
        "-i",
        "-e",
        "SW_URL",
        "skywalking-mcp:latest"
      ],
      "env": {
        "SW_URL": "http://localhost:12800"
      }
    }
  }
}
```

## Available Tools

SkyWalking MCP provides the following tools to query and analyze SkyWalking OAP data:

| Category    | Tool Name                | Description                            | Key Features                                                                                                                                                                                                                                                                  |
|-------------|--------------------------|----------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Trace**   | `get_trace_details`      | Get detailed trace information         | Retrieve trace by ID; **Multiple views**: `full` (complete trace), `summary` (overview with metrics), `errors_only` (error spans only); Detailed span analysis                                                                                                                |
| **Trace**   | `get_cold_trace_details` | Get trace details from cold storage    | Query historical traces from BanyanDB; **Multiple views**: `full`, `summary`, `errors_only`; Duration-based search; Historical incident investigation                                                                                                                         |
| **Trace**   | `query_traces`           | Query traces with intelligent analysis | Multi-condition filtering (service, endpoint, duration, state, tags); **Multiple views**: `full` (raw data), `summary` (intelligent analysis with performance insights), `errors_only` (error traces); Sort options; Slow trace detection; Performance metrics and statistics |
| **Metrics** | `query_single_metrics`   | Query single metric values             | Get specific metric values (CPM, response time, SLA, Apdex); Multiple entity scopes (Service, ServiceInstance, Endpoint, Process, Relations); Time range and cold storage support                                                                                             |
| **Metrics** | `query_top_n_metrics`    | Query top N metric rankings            | Rank entities by metric values; Configurable top N count; Ascending/descending order; Scope-based filtering; Performance analysis and issue identification                                                                                                                    |
| **Log**     | `query_logs`             | Query logs from SkyWalking OAP         | Filter by service, instance, endpoint, trace ID, tags; Time range queries; Cold storage support; Pagination support                                                                                                                                                           |
| **MQE**     | `execute_mqe_expression` | Execute MQE expressions for metrics    | Execute complex MQE (Metrics Query Expression) queries; Support calculations, aggregations, comparisons, TopN, trend analysis; Multiple result types (single value, time series, sorted list); Entity filtering and relation metrics; Debug and tracing capabilities          |
| **MQE**     | `list_mqe_metrics`       | List available metrics for MQE         | Discover available metrics for MQE queries; Filter by regex patterns; Get metric metadata (type, catalog); Support service, instance, endpoint, relation, database, and infrastructure metrics                                                                                |
| **MQE**     | `get_mqe_metric_type`    | Get metric type information            | Get detailed type information for specific metrics; Understand metric structure (regular value, labeled value, sampled record); Help with correct MQE expression syntax                                                                                                       |

## Contact Us

* Submit [an issue](https://github.com/apache/skywalking/issues/new) by using [MCP] as title prefix.
* Mail list: **dev@skywalking.apache.org**. Mail to `dev-subscribe@skywalking.apache.org`, follow the reply to subscribe
  the mail list.
* Join `skywalking` channel at [Apache Slack](http://s.apache.org/slack-invite). If the link is not working, find the
  latest one at [Apache INFRA WIKI](https://cwiki.apache.org/confluence/display/INFRA/Slack+Guest+Invites).
* Twitter, [ASFSkyWalking](https://twitter.com/ASFSkyWalking)

## License

[Apache 2.0 License.](/LICENSE)

[mcp]: https://modelcontextprotocol.io/