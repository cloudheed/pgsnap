# Contributing to pgsnap

Thank you for your interest in contributing to pgsnap!

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/pgsnap.git`
3. Create a branch: `git checkout -b feature/your-feature`
4. Make your changes
5. Run tests: `make test`
6. Run linter: `make lint`
7. Commit your changes: `git commit -m "Add your feature"`
8. Push to your fork: `git push origin feature/your-feature`
9. Open a Pull Request

## Development Setup

```bash
# Install Go 1.23+
# Clone the repo
git clone https://github.com/cloudheed/pgsnap.git
cd pgsnap

# Install dependencies
go mod tidy

# Install development tools
make tools

# Build
make build

# Test
make test

# Lint
make lint
```

## Code Style

- Follow standard Go conventions
- Run `make fmt` before committing
- Run `make lint` to check for issues
- Write tests for new functionality
- Keep commits focused and atomic

## Pull Request Guidelines

- Describe what your PR does and why
- Reference any related issues
- Ensure CI passes
- Keep PRs focused on a single change

## Reporting Issues

- Check existing issues first
- Include Go version, OS, and pgsnap version
- Provide steps to reproduce
- Include relevant logs or error messages

## License

By contributing, you agree that your contributions will be licensed under the Apache 2.0 License.
