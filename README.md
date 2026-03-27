## go_template
This repository is template for new Go projects.

## Features
*   **Linting:** Includes a `.golangci.yml` configuration for `golangci-lint` using T-Academia templat.
*   **Automated Tasks:** A `Makefile` provides convenient commands for building, testing, formatting, and linting the project.

## Prerequisites
*   `Go`
*   `golangci-lint`

## Getting Started
1.  **Initialize:** Create a new repository using this one as a template.
2.  **Clone:** Clone your newly created repository.
3.  **Update Module Path:** Edit the `go.mod` file (`go mod edit -module your-new-module-name`) and update the module declaration to match your project's path (e.g., `github.com/yourusername/yourprojectname`).
4.  **Adjust `TARGET`:** In the `Makefile`, change the default value of `TARGET ?=` to match the name of the directory inside `cmd/` that contains your main application (e.g., if your main app is in `cmd/myapp`, change it to `TARGET ?= myapp`). This determines the name of the built binary.
5.  **Customize:** Adapt the `.golangci.yml` if you need different linters or settings. Add your source code under `internal/` or `pkg/` and your main entry point in `cmd/`.

## Usage of 
The included `Makefile` simplifies common tasks:
*   `make build`: Builds the application binary and places it in `./bin/`.
*   `make test`: Runs all tests and prints coverage summary.
*   `make test_race`: Runs all tests with the race detector enabled.
*   `make html_test`: Generates an HTML coverage report (`coverage.html`) after running tests.
*   `make fmt`: Formats the entire codebase using `go fmt`.
*   `make lint`: Runs `golangci-lint` on the project.
*   `make clean`: Removes built binaries (`./bin/`) and coverage artifacts (`coverage.out`, `coverage.html`).