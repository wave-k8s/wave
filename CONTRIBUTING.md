# Contributing

To develop on this project, please fork the repo and clone into your `$GOPATH`.

Dependencies are **not** checked in so please download those separately.
Download the dependencies using [`dep`](https://github.com/golang/dep).

```bash
cd $GOPATH/src/github.com # Create this directory if it doesn't exist
git clone git@github.com:<YOUR_FORK>/wave wave-k8s/wave
go mod tidy
```

## Testing

Tests in Wave are run using a tool called [`ginkgo`](https://github.com/onsi/ginkgo).
The tests use [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) tooling
to run an etcd and kube-apiserver locally on during tests.

To prepare your machine for testing, run the `configure` script:

```
./configure
```

If you have any missing tools, there are `make` targets for setting up the testing
environment, run one of the following to set up Kubernetes environments for Kubernetes
versions 1.11, 1.12 and 1.13 respectively:

- `touch .env && make prepare-env-1.11`
- `touch .env && make prepare-env-1.12`
- `touch .env && make prepare-env-1.13`

This target is defined in [Makefile.tools](Makefile.tools) and we recommend that
you review the Makefile before you install the tooling.

## Pull Requests and Issues

We track bugs and issues using Github .

If you find a bug, please open an Issue.

If you want to fix a bug, please fork, fix the bug and open a PR back to this repo.
Please mention the open bug issue number within your PR if applicable.
