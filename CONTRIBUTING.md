# CONTRIBUTING.md

## Welcome to Orkestra! 🎼

First off, thank you for considering contributing to Orkestra. It's people like you that make Orkestra such a great tool.

**Orkestra** is a declarative Kubernetes operator runtime that aims to make operators accessible, composable, and observable — all without writing Go. Whether you're fixing a bug, adding a feature, improving documentation, or just asking questions, your help is appreciated.

This document provides guidelines for contributing. These are guidelines, not rules — use your best judgment, and feel free to propose changes to this document itself.

---

## Code of Conduct

This project and everyone participating in it is governed by the [Orkestra Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior to [Email](ialexeze@gmail.com).

---

## I Just Have a Question

> **Note:** Please don't file an issue to ask a question. You'll get faster results by using the resources below.

- **Read the docs** – Start with the [README](README.md) and [docs](./docs/) folder
- **Search issues** – Someone may have already asked your question
- **Kubernetes Slack** – Join the [#orkestra](https://kubernetes.slack.com) channel (coming soon)
- **GitHub Discussions** – Use [Discussions](https://github.com/ialexeze/orkestra/discussions) for questions

---

## How Can I Contribute?

### Reporting Bugs

#### Before Submitting a Bug Report

- **Check the [README](README.md)** – You might find that your "bug" is actually expected behavior
- **Search [issues](https://github.com/ialexeze/orkestra/issues)** – The bug may have already been reported
- **Check if it's a known limitation** – Some features are still in development

#### How to Submit a Good Bug Report

Bugs are tracked as [GitHub issues](https://github.com/ialexeze/orkestra/issues). Create an issue and provide the following information:

- **Use a clear, descriptive title**
- **Describe the exact steps to reproduce** – Be as specific as possible
- **Provide example YAML** – If the bug relates to a specific Katalog or CRD
- **Describe the behavior you observed** – And what you expected instead
- **Include logs** – Run with `--debug` flag and include output
- **Include environment details**:
  ```bash
  ork version
  kubectl version
  uname -a
  ```
- **If the bug is a crash**, include the stack trace

**Example bug report:**
```
Title: ork validate crashes when Komposer has empty sources block

Steps:
1. Create a file with:
   apiVersion: orkestra.konductor.io/v1Alpha
   kind: Komposer
   sources: {}
   
2. Run: ork validate --katalog ./test.yaml

Expected: Validation fails with clear error
Actual: Panics with "sources: unbound variable"
```

### Suggesting Enhancements

Enhancements are tracked as [GitHub issues](https://github.com/ialexeze/orkestra/issues). When suggesting an enhancement:

- **Use a clear, descriptive title**
- **Describe the problem** – What problem does this solve?
- **Describe the solution you'd like** – Be as specific as possible
- **Describe alternatives you've considered**
- **Provide examples** – Show how you'd use the feature in YAML

**Example enhancement:**
```
Title: Support S3 as a source type in Komposer

Problem: Our organization stores all infrastructure definitions in S3.
It would be great to source Katalogs directly from S3 buckets.

Solution: Add a new `sources.s3` field that accepts:
   - bucket
   - key
   - region
   - optional credentials via env vars

Example:
   sources:
     s3:
       - bucket: my-org-katalogs
         key: platform/crds.yaml
         region: us-east-1

Alternatives: We could use init containers to fetch from S3,
but native support would be cleaner.
```

### Improving Documentation

Documentation improvements are one of the most valuable contributions you can make. Orkestra's docs live in:

- `README.md` – Project overview, quick start
- `docs/` – Detailed guides and reference
- Code comments – Especially in `pkg/` and `internal/`

**Areas that always need help:**
- Fixing typos or unclear sentences
- Adding examples
- Clarifying error messages
- Translating to other languages
- Creating diagrams (Mermaid is preferred)

### Contributing Code

#### What to Work On

- **Good First Issues** – Look for issues labeled `good-first-issue`
- **Help Wanted** – Issues labeled `help-wanted`
- **Your Own Ideas** – But please open an issue first to discuss

#### Code Review Criteria

All contributions are reviewed for:
- **Correctness** – Does it work?
- **Test coverage** – Are there tests?
- **Documentation** – Is it documented?
- **Consistency** – Does it follow existing patterns?
- **Performance** – Will it scale?

---

## Development Setup

### Prerequisites

- Go 1.21 or higher
- `make` (optional, but helpful)
- Access to a Kubernetes cluster (for testing)

### Local Setup

```bash
# Clone the repository
git clone https://github.com/ialexeze/orkestra.git
cd orkestra

# Build the CLI
make build
# or
go build -o ork ./cmd/orkestra

# Run tests
make test
# or
go test ./...

# Run orkestra locally (with example)
./ork run --katalog examples/website/website-katalog.yaml
```

### Project Structure

```
orkestra/
├── cmd/           # CLI entry points
├── pkg/           # Public packages
│   ├── generate/  # Code generation
│   ├── kordinator/# Core controller cordination logic
│   ├── kubeclient/# Kubernetes client
│   ├── merger/    # Katalog merging
│   └── types/     # Shared types
├── examples/      # Example Katalogs
├── docs/          # Documentation
└── test/          # Integration tests
```

---

## Pull Request Process

### Before You Submit

1. **Run tests locally** – `make test`
2. **Run linters** – `make lint`
3. **Update documentation** – If you changed behavior
4. **Add tests** – If you added code

### Submitting the PR

1. **Fork the repository** and create your branch from `main`
2. **Use a descriptive branch name**: `feature/s3-sources`, `fix/validate-crash`
3. **Write a clear PR description** using the template
4. **Link related issues** – Use "Fixes #123" syntax
5. **Ensure CI passes** – All checks must be green

### PR Template

```markdown
# Description

Please include a summary of the change and which issue is fixed.

Fixes #(issue)

## Type of change

- [ ] Bug fix
- [ ] New feature
- [ ] Documentation update
- [ ] Refactoring

## How Has This Been Tested?

Please describe the tests you ran.

## Checklist

- [ ] My code follows the style guidelines
- [ ] I have added tests
- [ ] I have updated documentation
- [ ] All tests pass
```

### After Submitting

- **Respond to feedback** – Be responsive to review comments
- **Don't rebase after reviews** – Use merge commits instead
- **Be patient** – Maintainers are volunteers

---

## Style Guides

### Go Style Guide

Orkestra follows standard Go conventions:

- **Formatting** – `gofmt` (enforced by CI)
- **Linting** – `golangci-lint` with default config
- **Naming** – Follow [Go naming conventions](https://golang.org/doc/effective_go#names)
- **Comments** – Exported functions must be documented
- **Error handling** – Always check errors, use `%w` for wrapping
- **Tests** – Use table-driven tests where appropriate

**Example:**

```go
// LoadKatalog loads and validates a Katalog from the given path.
// Returns the parsed Katalog or an error.
func LoadKatalog(path string) (*Katalog, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("reading %q: %w", path, err)
    }
    
    var kat Katalog
    if err := yaml.Unmarshal(data, &kat); err != nil {
        return nil, fmt.Errorf("parsing %q: %w", path, err)
    }
    
    return &kat, nil
}
```

### Documentation Style Guide

- **Use Markdown** – All docs are Markdown
- **Use Mermaid** – For diagrams and graphs
- **Code blocks** – Specify language (```yaml, ```go, ```bash)
- **Headers** – Use ATX-style (`#`, `##`, `###`)
- **Lists** – Use `-` for unordered, `1.` for ordered
- **Links** – Use relative links within the repo

### Commit Message Guidelines

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

**Types:**
- `feat` – New feature
- `fix` – Bug fix
- `docs` – Documentation
- `style` – Formatting
- `refactor` – Code change that neither fixes a bug nor adds a feature
- `test` – Adding tests
- `chore` – Maintenance

**Examples:**
```
feat(sources): add support for S3 bucket sources

Add new sources.s3 field to Komposer that allows fetching
Katalogs directly from S3. Includes:
- S3 client with optional credentials
- Support for environment variables
- Integration tests

Closes #123
```

```
fix(validate): handle empty sources block without crashing

Previously, an empty sources: {} block would cause a panic.
Now it returns a validation error.

Fixes #456
```

---

## Community

### Communication Channels

- **GitHub Issues** – Bug reports, feature requests
- **GitHub Discussions** – Questions, ideas, show and tell
- **Kubernetes Slack** – [#orkestra](https://kubernetes.slack.com) (coming soon)

### Recognition

Contributors are recognized in:
- **CHANGELOG.md** – For significant changes
- **README.md** – Core contributors
- **Release notes** – For each release

### Roadmap

See [ROADMAP.md](./ROADMAP.md) for planned features and improvements.

---

## Thank You! ❤️

Your contributions, big or small, make Orkestra better for everyone. Whether you're fixing a typo, adding a feature, or just spreading the word — thank you.

**Happy orchestrating!** 🎼


