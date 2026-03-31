# Changes by Version

Release Notes.

## Next

### Security

* TLS certificate verification is now enforced for OAP connections. Added `--sw-insecure` flag to opt out (development/self-signed certs only).
* Sensitive fields (`authorization`, `password`, `token`, `secret`) are redacted in `--log-command` output.
* Environment variable references (`${VAR}`) in `--sw-username`/`--sw-password` now log a warning when the variable is not set, preventing silent unauthenticated requests.
* URL scheme validation rejects non-http/https OAP URLs.
* Regex patterns supplied to `list_mqe_metrics` are validated for complexity before compilation.

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

