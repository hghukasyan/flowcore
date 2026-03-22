# Contributing

Thanks for helping improve Flowcore. Small, focused changes are easier to review than large rewrites.

## Issues

Open an issue when you are unsure about an API change or a bigger feature. For clear bugs, a short repro (code or test) speeds things up.

## Pull requests

1. Fork the repo and create a branch from `main`.
2. Run checks locally:

   ```bash
   go fmt ./...
   go vet ./...
   go test ./... -race
   ```

3. Keep commits readable; match existing style and naming.
4. Add or update tests when behavior changes. Examples in `examples/` are optional but welcome for new user-facing features.
5. Describe what changed and why in the PR text.

## Scope

Flowcore is meant to stay small and embeddable. If a change pulls in new dependencies or large infrastructure-style code, it may be better as a separate module or a follow-up discussion first.

## License

By contributing, you agree your contributions are licensed under the same terms as the project ([LICENSE](LICENSE)).
