# Job Notes

Inspect `batch/v1 Job` status fields. Use to gate dependent resources on Job lifecycle conditions — cleanup tasks, chained workflows, migration guards.

## Reference

| Note | Signature | Returns |
|------|-----------|---------|
| `jobSucceeded` | `obj` | `bool` — at least one pod succeeded |
| `jobFailed` | `obj` | `bool` — at least one pod failed |
| `jobActive` | `obj` | `bool` — at least one pod currently running |

## Examples

```yaml
# Gate the application Deployment on a successful migration Job
when:
  - field: "{{ jobSucceeded .children.migrationJob }}"
    equals: "true"
  - field: "{{ jobFailed .children.migrationJob }}"
    equals: "false"

# Block creation while Job is still running
when:
  - field: "{{ jobActive .children.migrationJob }}"
    equals: "false"

# Status: surface job phase
- path: migrationStatus
  value: >
    {{ if jobSucceeded .children.migrationJob }}Complete
    {{ else if jobFailed .children.migrationJob }}Failed
    {{ else if jobActive .children.migrationJob }}Running
    {{ else }}Pending{{ end }}
```
