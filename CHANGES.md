# Changes by Version

Release Notes.

## 0.1.0

### Features

* Initial release of the `swmcp` binary (SkyWalking MCP server).
* Support for three MCP transport modes: `stdio`, `sse`, and `streamable`.
* Integration with Apache SkyWalking OAP via GraphQL, including:
  * Traces, logs, metrics, topology, alarms, and events query tools.
  * MQE (Metrics Query Extension) tools using the OAP `/graphql` endpoint.
* Prompt support for trace and log analysis and utility workflows.
* Embedded documentation and dynamic metrics resources for MQE.
* Makefile targets for build, lint, license checks, and Docker image creation.

