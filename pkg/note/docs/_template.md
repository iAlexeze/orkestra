# XX — Domain Title

One or two sentences introducing this domain of notes. Explain what these notes solve and why they exist.

## Reference

### `noteName`

A concise paragraph describing what the note does.  
This must be the first non‑empty paragraph after the heading — the generator uses it as the description.

Keywords: keyword1, keyword2, keyword3

```yaml
# Example usage of the note
# value: "{{ noteName .spec.field }}"
```

---

### `noteA` / `noteB`

Use this form when two notes share the same description and example.  
The generator will register both names with the same metadata.

Keywords: keywordA, keywordB, shared, multi-note

```yaml
# Example demonstrating both noteA and noteB
# value: "{{ noteA .spec.field }}"
# value: "{{ noteB .spec.field }}"
```

---

## Quick reference

| Note | Accepts | Returns | Use in |
|------|---------|---------|--------|
| `noteName` | input types | output type | typical usage |
| `noteA` | input types | output type | typical usage |
| `noteB` | input types | output type | typical usage |

