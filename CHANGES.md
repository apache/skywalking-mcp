# Changes by Version

Release Notes.

## 0.2.0

### Features

* TLS certificate verification is now enforced for OAP connections. Added `--sw-insecure` flag to opt out (development/self-signed certs only).
* Sensitive fields (`authorization`, `password`, `token`, `secret`) are redacted in `--log-command` output.
* Environment variable references (`${VAR}`) in `--sw-username`/`--sw-password` now log a warning when the variable is not set, preventing silent unauthenticated requests.
* URL scheme validation rejects non-http/https OAP URLs.
* Regex patterns supplied to `list_mqe_metrics` are validated for complexity before compilation.
* Added `--allowed-origins` flag to `sse` and `streamable` transports for CORS origin enforcement. When unset (default), any `Origin` is reflected back so all browser origins work out of the box. When set, only listed origins receive CORS headers; all others get `403 Forbidden`. Use `*` as an entry to send the wildcard header explicitly.
* Implement unit tests.
* Remove the unnecessary tool and parameter.
* Validate properties for tools.

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

