# Contributing to WebSwags

Thank you for your interest in contributing to WebSwags! We welcome contributions from the community. This document provides guidelines and information for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
- [Reporting Issues](#reporting-issues)
- [Submitting Pull Requests](#submitting-pull-requests)
- [Development Guidelines](#development-guidelines)
- [Testing](#testing)
- [Documentation](#documentation)

## Code of Conduct

This project follows a code of conduct to ensure a welcoming environment for all contributors. By participating, you agree to:

- Be respectful and inclusive
- Focus on constructive feedback
- Accept responsibility for mistakes
- Show empathy towards other contributors
- Help create a positive community

## Getting Started

### Prerequisites

- Go 1.25 or later
- Git
- Basic knowledge of Go programming

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:

   ```bash
   git clone https://github.com/YOUR_USERNAME/webswags.git
   cd webswags
   ```

3. Add the upstream remote:

   ```bash
   git remote add upstream https://github.com/Hossein-Roshandel/webswags.git
   ```

## Development Setup

1. Install dependencies:

   ```bash
   go mod download
   ```

2. Set up the development environment:

   ```bash
   ./setup-dev.sh
   ```

3. Verify the setup:

   ```bash
   make test
   make lint
   ```

## How to Contribute

### Types of Contributions

- **Bug fixes**: Fix issues in the codebase
- **Features**: Add new functionality
- **Documentation**: Improve documentation, README, etc.
- **Tests**: Add or improve test coverage
- **Code quality**: Refactoring, performance improvements

### Development Workflow

1. Choose an issue to work on or create a new one
2. Create a feature branch from `master`:

   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/issue-number-description
   ```

3. Make your changes
4. Run tests and linting:

   ```bash
   make test
   make lint
   ```

5. Commit your changes with clear commit messages
6. Push to your fork and create a pull request

## Reporting Issues

### Bug Reports

When reporting bugs, please include:

- **Description**: Clear description of the issue
- **Steps to reproduce**: Step-by-step instructions
- **Expected behavior**: What should happen
- **Actual behavior**: What actually happens
- **Environment**: Go version, OS, etc.
- **Logs**: Any relevant error messages or logs
- **Screenshots**: If applicable

### Feature Requests

For feature requests, please include:

- **Description**: What feature you'd like to see
- **Use case**: Why this feature would be useful
- **Proposed solution**: If you have ideas on implementation

## Submitting Pull Requests

### PR Guidelines

- **One feature per PR**: Keep PRs focused on a single change
- **Descriptive title**: Use a clear, descriptive title
- **Detailed description**: Explain what the PR does and why
- **Reference issues**: Link to any related issues
- **Tests**: Include tests for new functionality
- **Documentation**: Update documentation if needed

### PR Template

Please use the following template for your PR description:

```markdown
## Description
Brief description of the changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update
- [ ] Other

## Testing
Describe how you tested your changes

## Checklist
- [ ] Tests pass
- [ ] Linting passes
- [ ] Documentation updated
- [ ] Commit messages are clear
```

## Development Guidelines

### Code Style

- Follow Go conventions and idioms
- Use `gofmt` for formatting
- Write clear, readable code with comments
- Use meaningful variable and function names

### Commit Messages

Use clear, descriptive commit messages:

```
type(scope): description

[optional body]

[optional footer]
```

Examples:

- `feat: add support for OpenAPI 3.1`
- `fix: handle empty spec files gracefully`
- `docs: update installation instructions`

### Branch Naming

- `feature/description`: For new features
- `fix/description`: For bug fixes
- `docs/description`: For documentation changes
- `refactor/description`: For code refactoring

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-cover

# Run tests with race detector
make test-race
```

### Writing Tests

- Write unit tests for new functions
- Use table-driven tests where appropriate
- Test edge cases and error conditions
- Maintain good test coverage

## Documentation

### Code Documentation

- Add comments to exported functions and types
- Use Go doc conventions
- Keep comments up to date with code changes

### README Updates

- Update README.md for significant changes
- Add examples for new features
- Keep installation and usage instructions current

## Getting Help

If you need help:

- Check existing issues and documentation
- Ask questions in GitHub discussions
- Reach out to maintainers

## Recognition

Contributors will be recognized in the project's contributor list. Significant contributions may be highlighted in release notes.

Thank you for contributing to WebSwags! 🚀
