# Note Library — Developer Documentation

Notes are the template function library available in every Orkestra template expression — `status.fields`, `normalize.spec`, `onCreate`, `onReconcile`, mutation rules, and validation rules.

A note is a **pure, named transformation function**. Notes receive values and return transformed values. They cannot perform I/O, call external APIs, or produce side effects.

## How notes are used

In any Katalog template expression:

```yaml
status:
  fields:
    - path: phase
      value: "{{ boolTernary .spec.suspend \"Suspended\" \"Active\" }}"

normalize:
  spec:
    schedule: "{{ if typeMap .spec.schedule }}{{ cronFromMap .spec.schedule }}{{ else }}{{ cronNormalize .spec.schedule }}{{ end }}"

onCreate:
  secrets:
    - name: "{{ .metadata.name }}-creds"
      once: true
      data:
        password: "{{ randomAlphanumeric 32 }}"
```

## Documents

| File | Notes covered |
|------|---------------|
| [01-strings.md](01-strings.md) | `toLower` `toUpper` `trimSpace` `trim` `trimPrefix` `trimSuffix` `hasPrefix` `hasSuffix` `contains` `replace` `split` `join` `repeat` `camelToKebab` `truncate` |
| [02-math.md](02-math.md) | `add` `sub` `mul` `div` `mod` `min` `max` `clamp` `abs` |
| [03-conditional.md](03-conditional.md) | `ternary` `boolTernary` `boolDefault` `default` `coalesce` `empty` `notEmpty` |
| [04-types.md](04-types.md) | `typeOf` `typeMap` `typeList` `typeString` `typeNumber` `typeBool` `typeNull` `isEmpty` `len` `toInt` `toFloat` `toBool` `toString` |
| [05-cron.md](05-cron.md) | `cronExpr` `cronFromMap` `cronNormalize` `cronDescribe` `cronValid` `cronMinute` `cronHour` `cronDom` `cronMonth` `cronDow` `cronField` |
| [06-random.md](06-random.md) | `randomAlphanumeric` `randomHex` `randomBase64` |
| [07-collections.md](07-collections.md) | `listHas` `listGet` `listLen` `mapGet` `mapKeys` `mapValues` `asList` `asMap` `asString` |
| [08-safe-access.md](08-safe-access.md) | `getOr` `getStringOr` `getIntOr` `getBoolOr` |
| [09-kubernetes.md](09-kubernetes.md) | `meta` `labels` `annotations` `spec` `status` `get` `ownerKind` `ownerName` `hasCondition` |
| [10-container.md](10-container.md) | `containerImage` `containerEnv` `containerPort` |

Read the docs for the category you need. For a quick lookup of a specific function name, use the table above as an index.
