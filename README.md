Apache SkyWalking MCP
==========

<img src="http://skywalking.apache.org/assets/logo.svg" alt="Sky Walking logo" height="90px" align="right" />

**SkyWalking-MCP**: A [Model Context Protocol][mcp] (MCP) server for integrating AI agents with Skywalking OAP and the surrounding ecosystem.

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

## Contact Us
* Submit [an issue](https://github.com/apache/skywalking/issues/new) by using [MCP] as title prefix.
* Mail list: **dev@skywalking.apache.org**. Mail to `dev-subscribe@skywalking.apache.org`, follow the reply to subscribe the mail list.
* Join `skywalking` channel at [Apache Slack](http://s.apache.org/slack-invite). If the link is not working, find the latest one at [Apache INFRA WIKI](https://cwiki.apache.org/confluence/display/INFRA/Slack+Guest+Invites).
* Twitter, [ASFSkyWalking](https://twitter.com/ASFSkyWalking)

## License
[Apache 2.0 License.](/LICENSE)

[mcp]: https://modelcontextprotocol.io/