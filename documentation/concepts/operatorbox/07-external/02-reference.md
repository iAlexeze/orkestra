# Reference

## Wire format

```yaml
operatorBox:
  onReconcile:
    external:
      - name: healthCheck
        url: "{{ .spec.serviceUrl }}/health"
        method: GET
        body: ""
        token: "$API_TOKEN"
        headers:
          X-Request-ID: "{{ .metadata.name }}"
        timeout: 5s
        expectedStatus: 200
        continueOnError: false
        when:
          - field: status.phase
            notEquals: "Ready"
        anyOf: []
        sleep: ""
```

## Field reference

| Field | Required | Default | Description |
|---|---|---|---|
| `name` | yes | — | Identifier for accessing the result as `.external.<name>.*`. Must be a valid Go identifier — camelCase, no hyphens. |
| `url` | yes | — | Endpoint URL. Supports template expressions resolved against the current CR. |
| `method` | no | `GET` | HTTP method: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`. |
| `body` | no | `""` | Request body. Supports template expressions. Sets `Content-Type: application/json` automatically when non-empty. |
| `token` | no | `""` | Bearer token for the `Authorization` header. Use `$ENV_VAR` syntax — the value is expanded via `os.ExpandEnv` at call time. Never put raw secrets here. |
| `headers` | no | `{}` | Additional HTTP headers as a string map. Applied after `Authorization` and `Content-Type`. |
| `timeout` | no | `10s` | Per-call timeout. Go duration format: `"5s"`, `"1m"`, `"500ms"`. Set this to a fraction of your `resync:` period. |
| `expectedStatus` | no | `0` | When set, any response with a different status code is treated as a failure. When `0`: `2xx` succeeds, `4xx`/`5xx` fails. |
| `continueOnError` | no | `false` | `false`: a failed call halts the reconcile and writes `Ready=False`. `true`: failure is logged, `.error` is set, reconcile continues. |
| `when` | no | `[]` | AND conditions. If any condition fails, the call is skipped and `.called` is `"false"`. |
| `anyOf` | no | `[]` | OR conditions. At least one must pass. Combined with `when:` using AND semantics. |
| `sleep` | no | `""` | Delay injected before this call runs. Go duration format: `"2s"`. Use to pace sequential calls against rate-limited APIs, or to wait for an async side-effect from a previous call before polling. Not for production throttling — use `when:` conditions instead. |

## Result context

After a call completes, four fields are available under `.external.<name>`:

| Field | Type | Description |
|---|---|---|
| `.status` | `string` | HTTP status code as a string: `"200"`, `"404"`, `"503"`. Empty when the call failed before receiving a response. |
| `.body` | `string` | First 4096 bytes of the response body. Truncated silently for larger responses. |
| `.error` | `string` | Error message when the call failed. Empty string on success. |
| `.called` | `string` | `"true"` when the call ran; `"false"` when skipped because `when:`/`anyOf:` conditions failed. |

Access in templates:

```yaml
# Dot-path in a when: condition
- field: external.healthCheck.status
  equals: "200"

# Template expression in a value
value: "{{ .external.appConfig.body }}"

# Template expression in a later call's url or token
token: "{{ .external.tokenFetch.body }}"
```

## Constraints

**Body cap — 4096 bytes.** Large responses are truncated. If you need more, filter or paginate on the server side before the operator calls it.

**Sequential execution.** Calls run one at a time in declaration order. There is no parallelism. A later call can reference an earlier call's result; an earlier call cannot see a later one.

**camelCase names.** Call names must be valid Go identifiers. Hyphens in names break template access because `{{ .external.health-check.status }}` does not parse as a field access.

**Runs on every reconcile.** Unless gated by a `when:` condition, the call runs every time the CR is reconciled — including on every resync. Design calls to be idempotent or gate them with conditions.

**Token expansion via `os.ExpandEnv`.** The `token:` field is expanded using standard Go environment variable syntax: `$VAR` and `${VAR}`. This happens after template resolution, so `token: "${{ .spec.tokenEnvVar }}"` does not work — only static `$VAR` references expand.

**No response streaming.** The operator reads the full response body (up to 4096 bytes) before processing. Long-running or streaming endpoints are not supported.
