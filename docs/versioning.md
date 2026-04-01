# Versioning Strategy

qode follows [Semantic Versioning 2.0](https://semver.org/).

## Version Format

```
MAJOR.MINOR.PATCH-PRERELEASE+BUILD
```

- **MAJOR** — incompatible CLI or config changes
- **MINOR** — new features, backwards-compatible
- **PATCH** — bug fixes
- **PRERELEASE** — `alpha` or `beta` suffix
- **BUILD** — GitHub Actions run number (e.g., `+42`)

## Release Phases

| Phase | Version Range | Description |
|-------|--------------|-------------|
| **Alpha** | `0.1.x-alpha` | Active development. Breaking changes expected. |
| **Beta** | `0.x.x-beta` | Feature-complete. Bug fixes and stabilisation only. |
| **GA** | `1.0.0` | First stable release. |
| **Post-GA** | `1.x.x+` | Standard semver rules apply. |

### Phase Transitions

Transitions are manual. To move between phases, create and push a Git tag:

```bash
# Alpha release
git tag v0.1.0-alpha
git push origin v0.1.0-alpha

# Beta release (when feature-complete)
git tag v0.2.0-beta
git push origin v0.2.0-beta

# General availability
git tag v1.0.0
git push origin v1.0.0
```

### Incrementing Within a Phase

- Bug fix in alpha: `v0.1.0-alpha` → `v0.1.1-alpha`
- New feature in alpha: `v0.1.0-alpha` → `v0.2.0-alpha`
- Bug fix in beta: `v0.2.0-beta` → `v0.2.1-beta`
- Post-GA follows standard semver

## CI/CD Pipeline

### Every Merge to Main

1. Tests run (`go test ./...`)
2. Binaries built for all platforms via GoReleaser (snapshot mode)
3. A rolling `latest` pre-release on GitHub Releases is overwritten with fresh binaries
4. Version: `0.1.0-alpha+<run_number>` (e.g., `0.1.0-alpha+42`)

### Tagged Releases

When a version tag (`v*`) is pushed:

1. Tests run
2. GoReleaser creates a formal GitHub Release with changelog and binaries
3. The release is permanent and not overwritten

## Target Platforms

| OS | Architecture | Archive Format |
|----|-------------|----------------|
| Linux | x86_64 (amd64) | `.tar.gz` |
| Linux | ARM64 | `.tar.gz` |
| macOS | Intel (amd64) | `.tar.gz` |
| macOS | Apple Silicon (arm64) | `.tar.gz` |
| Windows | x86_64 (amd64) | `.zip` |
| Windows | ARM64 | `.zip` |

## How to Create a Release

1. Ensure `main` is in the desired state
2. Choose the version number following the rules above
3. Run the release script from the repo root:

```bash
tools/release.sh v0.1.0-alpha
```

The script checks that you are on `main`, that the working tree is clean, and that the tag does not already exist — then creates a lightweight tag and pushes it to `origin`.

4. GitHub Actions will build binaries and create the release automatically
5. Verify the release at https://github.com/nqode-io/qode/releases
