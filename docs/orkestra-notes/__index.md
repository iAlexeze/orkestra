# Orkestra Notes — Standard Library

Notes are pure transformation functions available in every `{{ }}` expression across a Katalog: conversion paths, status fields, mutation rules, `when:` conditions, `onCreate`, and `onReconcile`. They are the vocabulary of declarative operator behavior.

**Contract:** pure (same input → same output), safe (nil/empty input never panics, returns a safe zero value), stateless (no I/O, no side effects). Use [hooks](../runtime-manual/concepts/hooks.md) for anything that requires complex external calls not covered yet in orkestra.

---

| Domain | File | Notes |
|--------|------|-------|
| Cron | [cron.md](cron.md) | `cronToMap` `cronFromMap` `cronFromAny` `cronNormalize` `cronDescribe` `cronValid` `cronField` `cronMinute` `cronHour` `cronDom` `cronMonth` `cronDow` `cronExpr` |
| Semver | [semver.md](semver.md) | `semverMajor` `semverMinor` `semverPatch` `semverValid` `semverCompare` `semverBump` `semverConstraint` |
| String | [string.md](string.md) | `toLower` `toUpper` `trimSpace` `trim` `trimPrefix` `trimSuffix` `hasPrefix` `hasSuffix` `contains` `replace` `split` `join` `repeat` `camelToKebab` `truncate` |
| Math | [math.md](math.md) | `add` `sub` `mul` `div` `mod` `min` `max` `clamp` `abs` |
| Conditional | [conditional.md](conditional.md) | `ternary` `boolTernary` `boolDefault` `coalesce` `default` `empty` `notEmpty` |
| Type | [type.md](type.md) | `toInt` `toFloat` `toBool` `toString` `typeOf` `typeMap` `typeList` `typeString` `typeNumber` `typeBool` `typeNull` `len` `isEmpty` `isScalar` `isComposite` |
| Data | [data.md](data.md) | `toBase64` `fromBase64` `toJSON` `sha256sum` `slugify` `truncateName` |
| Random | [random.md](random.md) | `randomAlphanumeric` `randomHex` `randomBase64` |
| Time | [time.md](time.md) | `timeAgo` `timeSince` `timeFormat` `isExpired` `durationSeconds` `durationAdd` `durationValid` |
| Network | [network.md](network.md) | `cidrContains` `ipValid` `ipIsPrivate` |
| Resource quantities | [quantities.md](quantities.md) | `parseQuantity` `formatQuantity` `sumQuantity` |
| List / Map | [listmap.md](listmap.md) | `listHas` `listGet` `listLen` `mapGet` `mapKeys` `mapValues` `mapPick` `mapOmit` |
| Safe access | [safeaccess.md](safeaccess.md) | `getOr` `getStringOr` `getIntOr` `getBoolOr` |
| Type coercion | [coercion.md](coercion.md) | `asList` `asMap` `asString` |
| Kubernetes | [kubernetes.md](kubernetes.md) | `get` `meta` `name` `namespace` `labels` `annotations` `spec` `status` `phase` `resourceName` `resourceNamespace` `resourceUID` `resourceVersion` `creationTimestamp` `ownerKind` `ownerName` `hasCondition` `conditionReason` `conditionMessage` `resourceExists` `isTerminating` `generation` `observedGeneration` `isSynced` |
| Replicas | [replicas.md](replicas.md) | `replicasReady` `desiredReplicas` `readyReplicas` `availableReplicas` `updatedReplicas` |
| Service | [service.md](service.md) | `serviceClusterIP` `serviceNodePort` `serviceLoadBalancerIP` `serviceLoadBalancerHost` `endpointsReady` |
| Container | [container.md](container.md) | `containerImage` `containerEnv` `containerPort` `containerPortByName` `containerPorts` `containerCount` |
| Job | [job.md](job.md) | `jobSucceeded` `jobFailed` `jobActive` |

---

## Adding a note

1. Add the function to `pkg/note/<domain>.go`
2. Register it in that file's `xxxNotes()` FuncMap function
3. If it is a new file, register it in `buildNotes()` in `pkg/note/note.go`
4. Document it in the domain file above and update this index
5. Run `make build`

**Contract:** handle nil/empty input with a safe zero value, not a panic. Return `(value, error)` for functions that can meaningfully fail; return just `value` for infallible ones.
