# Changes by Version

Release Notes.

## 0.3.0

### Features

* Migrated to the official `modelcontextprotocol/go-sdk`, adding support for MCP protocol version `2026-07-28`. Clients on earlier revisions keep working: the server answers both the per-request `_meta` model and the legacy `initialize` handshake on the same endpoint.
* Tool input schemas are now inferred from their Go request types instead of being declared separately, so a schema can no longer drift from the struct it describes.

### Changes

* Removed the MQE documentation resources. The static documents were rarely read by clients, and `mqe://metrics/available` duplicated the `list_mqe_metrics` tool.
* `--log-command` now records the JSON-RPC message log provided by the SDK; its output format differs from previous releases. Sensitive-field redaction is preserved.
* Required tool parameters are now enforced: a call missing a required argument (for example `list_instances` without `start`) returns a validation error instead of silently falling back to a default time window.
* Requires Go 1.26 to build.

## 0.2.0

### Features

* TLS certificate verification is now enforced for OAP connections. Added `--sw-insecure` flag to opt out (development/self-signed certs only).
* Sensitive fields (`authorization`, `password`, `token`, `secret`) are redacted in `--log-command` output.
* Environment variable references (`${VAR}`) in `--sw-username`/`--sw-password` now log a warning when the variable is not set, preventing silent unauthenticated requests.
* URL scheme validation rejects non-http/https OAP URLs.
* Regex patterns supplied to `list_mqe_metrics` are validated for complexity before compilation.
* Added `--allowed-origins` flag to `sse` and `streamable` transports for CORS origin enforcement. When unset (default), any `Origin` is reflected back so all browser origins work out of the box. When set, only listed origins receive CORS headers; all others get `403 Forbidden`. Use `*` as an entry to send the wildcard header explicitly.
* Increased reliability of core CLI commands through expanded automated test coverage.
* Removed an unused CLI tool and its associated parameter to simplify the interface and avoid confusion.
* Added validation for tool configuration properties, returning clear errors when required values are missing or invalid.

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

