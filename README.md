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
  -h, --help                 help for swmcp
      --log-command          When true, log commands to the log file
      --log-file string      Path to log file
      --log-level string     Logging level (debug, info, warn, error) (default "info")
      --read-only            Restrict the server to read-only operations
      --sw-url string        Specify the OAP URL to connect to (e.g. http://localhost:12800)
      --sw-username string   Username for basic auth to SkyWalking OAP (supports ${ENV_VAR} syntax)
      --sw-password string   Password for basic auth to SkyWalking OAP (supports ${ENV_VAR} syntax)
  -v, --version              version for swmcp

Use "swmcp [command] --help" for more information about a command.
```

You could start the MCP server with the following command:

```bash
# use stdio server
bin/swmcp stdio --sw-url http://localhost:12800

# with basic auth (raw password)
bin/swmcp stdio --sw-url http://localhost:12800 --sw-username admin --sw-password admin

# with basic auth (password from environment variable)
bin/swmcp stdio --sw-url http://localhost:12800 --sw-username admin --sw-password '${SW_PASSWORD}'

# or use SSE server
bin/swmcp sse --sse-address localhost:8000 --base-path /mcp --sw-url http://localhost:12800
```

### Usage with Cursor, Copilot, Claude Code

```json
{
  "mcpServers": {
    "skywalking": {
      "command": "swmcp stdio",
      "args": [
        "--sw-url", "http://localhost:12800",
        "--sw-username", "admin",
        "--sw-password", "${SW_PASSWORD}"
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
		"skywalking-mcp:latest",
		"--sw-url",
		"http://localhost:12800"
      ]
    }
  }
}
```

## Available Tools

SkyWalking MCP provides the following tools to query and analyze SkyWalking OAP data:

| Category     | Tool Name                      | Description                                                                                       |
|--------------|--------------------------------|---------------------------------------------------------------------------------------------------|
| **Session**  | `set_skywalking_url`           | Set the SkyWalking OAP server URL and optional basic auth credentials for the current session. Supports `${ENV_VAR}` syntax for credentials. |
| **Trace**    | `query_traces`                 | Query traces with multi-condition filtering (service, endpoint, state, tags, and time range via start/end/step). Supports `full`, `summary`, and `errors_only` views with performance insights. |
| **Log**      | `query_logs`                   | Query logs with filters for service, instance, endpoint, trace ID, tags, and time range. Supports cold storage and pagination. |
| **MQE**      | `execute_mqe_expression`       | Execute MQE (Metrics Query Expression) to query and calculate metrics data. Supports calculations, aggregations, TopN, trend analysis, and multiple result types. |
| **MQE**      | `list_mqe_metrics`             | List available metrics for MQE queries. Filter by regex pattern; returns metric name, type, and catalog. |
| **MQE**      | `get_mqe_metric_type`          | Get type information (REGULAR_VALUE, LABELED_VALUE, SAMPLED_RECORD) for a specific metric to help build correct MQE expressions. |
| **Metadata** | `list_layers`                  | List all layers registered in SkyWalking OAP (e.g. GENERAL, MESH, K8S).                          |
| **Metadata** | `list_services`                | List all services registered in SkyWalking OAP under a specific layer.                            |
| **Metadata** | `list_instances`               | List all instances of a service (e.g. pods or JVM processes).                                     |
| **Metadata** | `list_endpoints`               | List endpoints of a service with optional keyword filtering.                                       |
| **Metadata** | `list_processes`               | List processes of a service instance.                                                              |
| **Event**    | `query_events`                 | Query events (deployments, restarts, scaling) with filters for service, instance, endpoint, type, and layer. |
| **Alarm**    | `query_alarms`                 | Query alarms triggered by metric threshold breaches. Filter by scope, keyword, and tags.          |
| **Topology** | `query_services_topology`      | Query global or scoped service topology. Optionally filter by specific service IDs or layer.      |
| **Topology** | `query_instances_topology`     | Query service instance topology between a client service and a server service.                    |
| **Topology** | `query_endpoints_topology`     | Query endpoint dependency topology for a given endpoint.                                          |
| **Topology** | `query_processes_topology`     | Query process topology for a given service instance.                                              |

## Available Prompts

SkyWalking MCP provides the following prompts for guided analysis workflows:

| Category        | Prompt Name                  | Description                                                                                          | Arguments                                                                                    |
|-----------------|------------------------------|------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------|
| **Performance** | `analyze-performance`        | Analyze service performance using metrics tools                                                      | `service_name` (required), `start` (optional), `end` (optional)                             |
| **Performance** | `compare-services`           | Compare performance metrics between multiple services                                                | `services` (required), `metrics` (optional), `start` (optional), `end` (optional)           |
| **Performance** | `top-services`               | Find top N services ranked by a given metric                                                         | `metric_name` (required), `top_n` (optional), `order` (optional)                            |
| **Trace**       | `investigate-traces`         | Investigate traces for errors and performance issues                                                 | `service_id` (optional), `trace_state` (optional), `start` (optional), `end` (optional)     |
| **Trace**       | `trace-deep-dive`            | Deep dive analysis of a specific trace                                                               | `trace_id` (required), `view` (optional)                                                    |
| **Log**         | `analyze-logs`               | Analyze service logs for errors and patterns                                                         | `service_id` (optional), `log_level` (optional), `start` (optional), `end` (optional)       |
| **Topology**    | `explore-service-topology`   | Explore services, instances, endpoints, and processes within a layer and time range                  | `layer` (required), `start` (required), `end` (optional)                                    |
| **MQE**         | `build-mqe-query`            | Help build MQE expressions for complex metric queries                                                | `query_type` (required), `metrics` (required), `conditions` (optional)                      |
| **MQE**         | `explore-metrics`            | Explore available metrics and their types                                                            | `pattern` (optional), `show_examples` (optional)                                            |
| **Utility**     | `generate_duration`          | Convert a natural-language time range into a `{start, end}` duration object for use with other tools | `time_range` (required)                                                                      |

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