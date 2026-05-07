# 01 — Project Detection

`Detect(dir string)` is the entry point for everything the doctor package does. It examines the project directory and returns a `*ProjectInfo` that all other functions consume.

## ProjectInfo

```go
type ProjectInfo struct {
    Dir           string
    HasDockerfile bool
    GitCommit     string   // short SHA, e.g. "a3f5c2b"
    Language      Language
    LangMarker    string   // file that triggered detection, e.g. "go.mod"
    Port          string   // from PORT in .env, or language default
    EnvVars       []EnvVar // all parsed .env variables
    Secrets       []EnvVar // IsCfg == false
    Config        []EnvVar // IsCfg == true
    HasFrontend   bool
    AppName       string   // derived from directory basename
}
```

A missing `.env` is not an error — `EnvVars`, `Secrets`, and `Config` are simply empty.

## Language detection

`detectLanguage` checks for known marker files in the project root, in this order:

| Marker file | Language |
|-------------|----------|
| `go.mod` | Go |
| `package.json` | Node.js |
| `pom.xml` | Java |
| `requirements.txt` | Python |
| `Gemfile` | Ruby |
| `Cargo.toml` | Rust |

The first match wins. `LangMarker` records which file triggered the match — this is what `ork doctor` displays to the user:

```
✓ Language: Go  (go.mod)
```

## Port detection

Port is read from `PORT` in `.env`. If not present, a language default is used:

| Language | Default |
|----------|---------|
| Go | 8080 |
| Node.js | 3000 |
| Java | 8080 |
| Python | 8000 |
| Ruby | 3000 |
| Rust | 8080 |
| Unknown | 8080 |

## Frontend detection

`HasFrontend` is true when any of the following is found:

- A `build/`, `dist/`, or `public/` directory exists in the project root.
- The language is Node.js and `package.json` contains a known framework name: `react`, `vue`, `angular`, `next`, `nuxt`, or `svelte`.

When `HasFrontend` is true, the generator includes an Ingress in the Katalog and a `host` field in `app.yaml`.

## Git commit

`shortGitCommit` reads `.git/HEAD` directly — no `git` binary required. It follows symbolic refs (`ref: refs/heads/main`) to the commit SHA and returns the first 7 characters. Returns `""` for detached HEAD with a short SHA and for non-git directories.

The short SHA is used as the default image tag in `ork deploy`:

```
ghcr.io/myorg/my-app:a3f5c2b
```

## Usage

```go
info, err := doctor.Detect(".")
if err != nil {
    return err
}

fmt.Println(info.Name)   // "my-app"
fmt.Println(info.Language)  // "Go"
fmt.Println(info.Port)      // "8080"
fmt.Println(info.GitCommit) // "a3f5c2b"
```

→ Next: [02-envfile.md](02-envfile.md)
