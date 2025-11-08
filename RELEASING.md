# Release Process

This document explains how releases are created for the go-wztonx-converter project.

## Automated Releases with GoReleaser

This project uses [GoReleaser](https://goreleaser.com/) and GitHub Actions to automatically build and publish releases.

## Creating a New Release

To create a new release:

1. **Ensure all changes are committed and pushed to main**
   ```bash
   git add .
   git commit -m "Your commit message"
   git push origin main
   ```

2. **Create and push a version tag**
   ```bash
   # Create a tag following semantic versioning (e.g., v1.0.0, v1.2.3)
   git tag -a v1.0.0 -m "Release v1.0.0"
   
   # Push the tag to GitHub
   git push origin v1.0.0
   ```

3. **GitHub Actions will automatically:**
   - Run all tests
   - Build binaries for multiple platforms:
     - Linux (amd64, arm64, arm)
     - macOS (amd64, arm64)
     - Windows (amd64)
   - Create archives (tar.gz for Unix, zip for Windows)
   - Generate checksums
   - Create a GitHub release with:
     - Changelog
     - Release notes
     - All binaries attached
   - Inject version information into binaries

## Release Workflow

The release process is defined in `.github/workflows/release.yml`:

- **Trigger**: Pushing a tag matching `v*.*.*` pattern
- **Steps**:
  1. Checkout code
  2. Set up Go environment
  3. Run tests
  4. Execute GoReleaser

## GoReleaser Configuration

The release configuration is in `.goreleaser.yaml`:

### Build Configuration
- **Binaries**: Cross-compiled for multiple OS/architecture combinations
- **Optimization**: Stripped binaries with `-s -w` flags
- **Version Info**: Injects version, commit, and build date into the binary

### Archive Configuration
- **Unix**: `.tar.gz` format
- **Windows**: `.zip` format
- **Contents**: Binary + README + USAGE + LICENSE

### Release Notes
- Auto-generated changelog
- Grouped by commit type (features, fixes, etc.)
- Excludes merge commits and non-user-facing changes

## Manual Testing Before Release

Before creating a release tag, test locally:

```bash
# Run tests
go test -v ./...

# Build for your platform
go build

# Test the binary
./wztonx-converter --help

# Optional: Test with goreleaser locally (requires goreleaser installed)
goreleaser release --snapshot --clean
```

## Version Numbering

Follow [Semantic Versioning](https://semver.org/):

- **MAJOR** (v2.0.0): Breaking changes
- **MINOR** (v1.1.0): New features, backwards compatible
- **PATCH** (v1.0.1): Bug fixes, backwards compatible

Examples:
- `v0.1.0` - Initial development release
- `v1.0.0` - First stable release
- `v1.1.0` - Added new feature
- `v1.1.1` - Fixed a bug

## Release Checklist

Before creating a release:

- [ ] All tests pass
- [ ] Update CHANGELOG.md (if maintained separately)
- [ ] Update version-related documentation
- [ ] Test building on your local machine
- [ ] Verify no sensitive data in commits
- [ ] Tag follows semantic versioning

## Viewing Releases

- Releases are visible on the [GitHub Releases page](https://github.com/ErwinsExpertise/go-wztonx-converter/releases)
- Each release includes:
  - Changelog
  - Pre-built binaries for all platforms
  - Checksums for verification
  - Installation instructions

## Troubleshooting

### Release Failed
1. Check the Actions tab in GitHub for error logs
2. Common issues:
   - Tests failing
   - Build errors on specific platforms
   - GoReleaser configuration errors

### Re-releasing a Version
If a release has issues:
1. Delete the tag locally and remotely:
   ```bash
   git tag -d v1.0.0
   git push --delete origin v1.0.0
   ```
2. Delete the GitHub Release
3. Fix the issues
4. Create the tag again

## CI/CD Pipeline

In addition to releases, the project has a CI workflow (`.github/workflows/ci.yml`) that:
- Runs on every push and pull request
- Tests on multiple OS and Go versions
- Runs linting with golangci-lint
- Uploads test coverage
- Builds for multiple platforms

This ensures quality before releases are created.
