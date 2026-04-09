# Imagine this nested map

Here’s a realistic Kubernetes‑style structure:

```go
m := map[string]interface{}{
  "status": map[string]interface{}{
    "phase": "Running",
    "conditions": map[string]interface{}{
      "Ready": "True",
    },
  },
  "metadata": map[string]interface{}{
    "name": "my-app",
  },
}
```

Now let’s walk through a path:

```
"status.conditions.Ready"
```

---

## Step‑by‑step visual walkthrough

We start with:

```
current = m
path = "status.conditions.Ready"
parts = ["status", "conditions", "Ready"]
```

---

#### 🔹 1. First part: `"status"`

```
current = m
```

Is `current` a map?  
Yes → continue.

Look up `"status"`:

```
current = m["status"]
```

Which is:

```go
map[string]interface{}{
  "phase": "Running",
  "conditions": map[string]interface{}{
    "Ready": "True",
  },
}
```

---

#### 🔹 2. Second part: `"conditions"`

Now:

```
current = m["status"]
```

Is `current` a map?  
Yes → continue.

Look up `"conditions"`:

```
current = m["status"]["conditions"]
```

Which is:

```go
map[string]interface{}{
  "Ready": "True",
}
```

---

#### 🔹 3. Third part: `"Ready"`

Now:

```
current = m["status"]["conditions"]
```

Is `current` a map?  
Yes → continue.

Look up `"Ready"`:

```
current = m["status"]["conditions"]["Ready"]
```

Which is:

```
"True"
```

---

### 4. Loop ends — we reached the final value

`current = "True"`

Check:

- Is it nil? → no  
- Is it a string? → yes  

So return:

```
"True"
```

---

#### Final result

```go
NavigateDotPath(m, "status.conditions.Ready") 
→ "True"
```

---

## Another example: `"metadata.name"`

Path:

```
["metadata", "name"]
```

Walk:

1. `current = m["metadata"]`
2. `current = m["metadata"]["name"]`
3. `"my-app"` → return `"my-app"`

---

### Example where something is missing

Path:

```
"status.notThere.value"
```

Walk:

1. `current = m["status"]` → OK  
2. `current = m["status"]["notThere"]` → ❌ missing  

Return:

```
""
```

No panic, no error — just empty string.

---

## Why this function is so useful

It gives you:

- Safe navigation  
- No panics  
- No type assertions needed  
- Works with arbitrary JSON/YAML  
- Perfect for templates and UI rendering  

It’s basically a tiny, safe version of:

```
m.status.conditions.Ready
```

But for dynamic maps.
