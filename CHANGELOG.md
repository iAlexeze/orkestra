# Changelog

### CI/CD & Helm Chart Improvements

#### 🔧 Helm Chart
- Removed unnecessary configurations after testing
- Chart now uses stripped image tag (without `v` prefix) for `appVersion` and container image tags
- Updated `values.yaml` to reference the correct image tag format

#### ⚙️ CI/CD Workflows – Made Fully Reusable
- All workflows now accept configurable inputs:
  - `image_tag` (stripped version) passed from a central `prepare` job
  - Repository, image names, Helm repo URL, Homebrew tap, etc.
- Removed hardcoded project names (`orkspace`, `ialexeze`, `orkestra`) – workflows are portable
- Added `prepare` job to strip `v` from Git tags and propagate `image_tag` to downstream jobs
- Standardized tag handling: Git tags keep `v`, container images and Helm charts use plain semver

#### 🚀 Final Release Workflow
- Orchestrates all jobs with proper dependencies and conditionals
- Uses `prepare` to compute metadata once
- All reusable workflows called with explicit inputs (defaults applied in the called workflows)
- Added release summary job that aggregates status from all components

#### 📦 Affected Workflows
- `build-matrix.yml` (unchanged but now receives metadata via prepare)
- `build-push-images.yml` – configurable image names & registry
- `package-examples.yml` – configurable repo and project name
- `release-helm.yml` – configurable chart name, repo URL, namespace
- `sign-and-release.yml` – fully configurable (GitHub repo, container registry, Helm repo, Homebrew tap)
- `publish-homebrew.yml` – accepts tap repository and main repo
- `release-summary.yml` – uses same configurable inputs for success instructions