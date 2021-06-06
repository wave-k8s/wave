module github.com/k8s-wave/wave

go 1.16

require (
	github.com/go-logr/glogr v0.1.0
	github.com/kubernetes-sigs/kubebuilder v0.1.11-0.20180607060409-29bccabffb06
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.10.1
	github.com/prometheus/client_golang v1.7.1
	github.com/spf13/pflag v1.0.5
	github.com/wave-k8s/wave v0.5.0
	golang.org/x/sys v0.0.0-20210603125802-9665404d3644 // indirect
	golang.org/x/tools v0.1.2 // indirect
	k8s.io/api v0.21.1
	k8s.io/apiextensions-apiserver v0.18.2 // indirect
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/code-generator v0.21.1
	k8s.io/kube-aggregator v0.21.1 // indirect
	sigs.k8s.io/controller-runtime v0.2.2
)

replace k8s.io/client-go => k8s.io/client-go v0.21.1
