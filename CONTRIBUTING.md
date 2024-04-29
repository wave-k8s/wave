# Contributing

To develop on this project, please fork and clone the repo.

You can check dependencies using:

```bash
make tidy
```

## Testing

The tests use [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) tooling
to run an etcd and kube-apiserver locally on during tests.

You can install dependencies (into `$PWD/bin`) using:

```bash
make envtest
```

Tests in Wave are run using `go test` (which can be used to run them selectively).

You can run the full test suite locally using:

```bash
make test
```

You can run the linter locally using:

```bash
make lint
```

If you want to debug/test github actions locally you can use [act](https://github.com/nektos/act).
Follow the [act install instructions](https://nektosact.com/installation/index.html).
Then simply run tests using:

```bash
act
```

## Pull Requests and Issues

We track bugs and issues using Github .

If you find a bug, please open an Issue.

If you want to fix a bug, please fork, fix the bug and open a PR back to this repo.
Please mention the open bug issue number within your PR if applicable.
