# Contributing to `pkg/resources`

Thank you for your interest in improving Orkestra’s internal resources!

## What to Contribute

- **New resource types** – e.g., `HorizontalPodAutoscaler`, `NetworkPolicy`, `Ingress`.
- **Improvements** – bug fixes, performance optimisations, better error messages.
- **Test coverage** – unit tests for existing or new resources.

## Development Setup

1. Clone the Orkestra repository.
2. Run `make test` to run all tests.
3. Make your changes.
4. Add tests for new functionality.
5. Run `make lint` to ensure code style.
6. Submit a pull request.

## Adding a New Resource Type

Follow the steps outlined in the [README](./README.md). Make sure to:

- Add the template source type to `pkg/types/hooks.go`.
- Implement the four functions in your subdirectory.
- Add a resolver method in `template/resolver.go`.
- Add a runner in `pkg/reconciler/`.
- Wire the runner in `generic.go`.
- Write unit tests covering all functions.

## Code Style

- Follow standard Go conventions (`gofmt`, `golint`).
- Use the same function signatures as existing resources.
- Use `kubeclient.Kubeclient` for all API calls; never use a direct client.
- Return wrapped errors with context (`fmt.Errorf("creating deployment: %w", err)`).

## Testing

- Each resource must have a test file with at least:
  - Test for `Create` (resource created, already exists).
  - Test for `Update` (drift detection, update, creation).
  - Test for `Delete` (resource exists, not exists).
  - Test for `Resolve` (defaults, labels).

## Pull Request Process

1. Open a pull request with a clear title and description.
2. Link any related issues.
3. Ensure CI passes.
4. Wait for review. We’ll respond within a few days.

## Questions?

Open a [GitHub Discussion](https://github.com/orkestra-sh/orkestra/discussions) or reach out on the Kubernetes Slack (#orkestra channel).