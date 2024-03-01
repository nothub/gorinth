# Contributing

## Build

To build only binaries, run:

```sh
goreleaser build --clean --snapshot
```

## Release

To build a snapshot release, run:

```sh
goreleaser release --clean --snapshot
```

To build and publish a full release, run:

```sh
git tag v0.1.0 && git push origin v0.1.0
goreleaser release --clean --fail-fast
```