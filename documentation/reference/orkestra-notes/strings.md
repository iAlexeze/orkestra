# String Notes

String notes manipulate text values. They cover the cases that come up constantly when building resource names, labels, and field values from CR data.

## Reference

| Note | Description |
|------|-------------|
| `toLower` | Convert a string to all lowercase or all uppercase. |
| `toUpper` | Convert a string to all lowercase or all uppercase. |
| `trimSpace` | Remove leading and trailing whitespace. |
| `trim` | Remove all leading and trailing occurrences of a cutset string. |
| `trimPrefix` | Remove a specific prefix or suffix if present. |
| `trimSuffix` | Remove a specific prefix or suffix if present. |
| `hasPrefix` | Return `true` if the string starts or ends with the given substring. |
| `hasSuffix` | Return `true` if the string starts or ends with the given substring. |
| `contains` | Return `true` if the string contains the substring. |
| `replace` | Replace all occurrences of `old` with `new` in `s`. |
| `split` | Split a string by a separator into a slice. |
| `join` | Join a slice of strings into a single string with a separator. |
| `repeat` | Repeat a string n times. |
| `camelToKebab` | Convert CamelCase or PascalCase to kebab-case. |
| `truncate` | Truncate a string to at most `n` characters. |

## Examples

```yaml
# toLower
# value: "{{ toLower .spec.environment }}"
# production → production
# PRODUCTION → production

# value: "{{ toUpper .spec.tier }}"
# premium → PREMIUM

# toUpper
# value: "{{ toLower .spec.environment }}"
# production → production
# PRODUCTION → production

# value: "{{ toUpper .spec.tier }}"
# premium → PREMIUM

# trimSpace
# value: "{{ trimSpace .spec.domain }}"
# "  example.com  " → "example.com"

# trim
# value: "{{ trim .spec.tag \"/\" }}"
# "/images/nginx/" → "images/nginx"

# trimPrefix
# value: "{{ trimPrefix .metadata.name \"app-\" }}"
# "app-frontend" → "frontend"

# value: "{{ trimSuffix .spec.image \":latest\" }}"
# "nginx:latest" → "nginx"

# trimSuffix
# value: "{{ trimPrefix .metadata.name \"app-\" }}"
# "app-frontend" → "frontend"

# value: "{{ trimSuffix .spec.image \":latest\" }}"
# "nginx:latest" → "nginx"

# hasPrefix
# value: "{{ hasPrefix .spec.image \"gcr.io/\" }}"
# "gcr.io/myproject/app:v1" → true

# hasSuffix
# value: "{{ hasPrefix .spec.image \"gcr.io/\" }}"
# "gcr.io/myproject/app:v1" → true

# contains
# value: "{{ contains .spec.environment \"prod\" }}"
# "production" → true
# "staging"    → false

# replace
# value: "{{ replace .spec.domain \".\" \"-\" }}"
# "my.app.example.com" → "my-app-example-com"

# split
# Extract the first tag from a comma-separated list
# value: "{{ index (split .spec.tags \",\") 0 }}"
# spec.tags: "latest,stable,v2" → "latest"

# join
# value: "{{ join .spec.hosts \", \" }}"
# ["api.example.com", "admin.example.com"] → "api.example.com, admin.example.com"

# repeat
# value: "{{ repeat \"ha\" 3 }}"
# → "hahaha"

# camelToKebab
# value: "{{ camelToKebab .spec.controllerName }}"
# "WebsiteOperator" → "website-operator"
# "myAppName"       → "my-app-name"

# truncate
# value: "{{ truncate .metadata.name 63 }}"
# Names longer than 63 chars are clipped and get "..." appended.
# Names at or under 63 chars are returned unchanged.
```
