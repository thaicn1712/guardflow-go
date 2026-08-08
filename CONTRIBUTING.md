# Contributing

```bash
git clone https://github.com/thaicn1712/guardflow-go
cd guardflow-go
go test ./...
go vet ./...
gofmt -l .   # must print nothing
```

All three must pass before a PR is merged — CI runs the same checks. New public API needs a test alongside it (`*_test.go`).
