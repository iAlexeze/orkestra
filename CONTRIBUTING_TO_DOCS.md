# Contributing to Orkestra Documentation

Thank you for helping improve the Orkestra documentation.  
Clear, thoughtful docs are a core part of the Orkestra experience — and contributions are always welcome.

This guide explains how to contribute new documentation, improve existing pages, and follow the project’s documentation standards.

---

## 🧭 Documentation Philosophy

Orkestra documentation is built around three principles:

### 1. **Clarity over completeness**
Explain concepts simply. Avoid jargon unless necessary. Prefer examples over theory.

### 2. **Structure over sprawl**
Every doc belongs in one of four categories:
- **Concepts** — what Orkestra *is*
- **Guides** — how to *use* Orkestra
- **Reference** — detailed schemas and APIs
- **Internals** — how Orkestra *works under the hood*

### 3. **Consistency over creativity**
Use the same tone, formatting, and structure across all docs.

---

## 📂 Documentation Structure

All documentation lives under the `docs/` directory:

```
docs/
  start-here.md
  katalog.md
  komposer.md
  cli.md
  ...
  orkestra-registry/
    orkestra-registry-vision.md
    orkestra-registry-technical-documentation.md
```

The index page (`docs/README.md`) organizes everything into:

- Concepts  
- Guides  
- Reference  
- Internals  
- OrkestraRegistry  
- Publications  

When adding a new doc, place it in the correct category and update the index.

---

## ✍️ Writing Guidelines

### Tone
- Friendly, direct, and confident  
- Avoid passive voice  
- Prefer short sentences  
- Use examples generously  

### Formatting
- Use Markdown headings (`##`, `###`) consistently  
- Use fenced code blocks for YAML, Go, CLI commands  
- Use tables for structured data  
- Use callouts sparingly (e.g., **Note**, **Warning**)  

### Examples
Every concept should include at least one example.  
Every guide should include a runnable snippet.

---

## 🧪 Testing Your Documentation

Before submitting:

- Run through your examples manually  
- Ensure all links resolve  
- Validate YAML with `kubectl apply --dry-run=client -f`  
- Check spelling and grammar  

---

## 🔀 Submitting a Documentation PR

1. Create a new branch:
   ```bash
   git checkout -b docs/<topic>
   ```

2. Add or update documentation under `docs/`.

3. Update `docs/README.md` if needed.

4. Commit with a clear message (see template below).

5. Open a Pull Request using the PR template below.

---

## 📝 Commit Message Template

```
docs: add <topic> documentation

- Added <file> under docs/<category>
- Updated docs/README.md index
- Included examples and cross-links
```

Examples:
```
docs: add OrkestraRegistry Vision document
docs: improve Katalog reference examples
docs: add Start Here onboarding guide
```

---

## 🔧 Pull Request Template

```
## Summary
Describe what this PR adds or improves in the documentation.

## Changes
- Added <new doc>
- Updated index
- Improved examples / formatting / clarity

## Category
- [ ] Concepts
- [ ] Guides
- [ ] Reference
- [ ] Internals
- [ ] OrkestraRegistry
- [ ] Publications

## Checklist
- [ ] All links resolve
- [ ] Examples tested
- [ ] Markdown lint passes
- [ ] Index updated (docs/README.md)
```

---

## 🙌 Thank You

Documentation is how Orkestra becomes accessible to everyone — platform engineers, SREs, and teams who want operators without writing operators.

Your contribution helps shape the future of Kubernetes extensibility.
