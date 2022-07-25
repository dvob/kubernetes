// This is a generated file. Do not edit directly.

module k8s.io/sample-apiserver

go 1.16

require (
	cloud.google.com/go v0.81.0 // indirect
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/google/gofuzz v1.1.0
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/cobra v1.4.0
	google.golang.org/genproto v0.0.0-20220107163113-42d7afdf6368 // indirect
	k8s.io/apimachinery v0.0.0
	k8s.io/apiserver v0.24.3
	k8s.io/client-go v0.0.0
	k8s.io/code-generator v0.0.0
	k8s.io/component-base v0.0.0
	k8s.io/kube-openapi v0.0.0-20220328201542-3ee0da9b0b42
	k8s.io/utils v0.0.0-20220210201930-3a6ce19ff2f9
)

replace (
	k8s.io/api => ../api
	k8s.io/apimachinery => ../apimachinery
	k8s.io/apiserver => ../apiserver
	k8s.io/client-go => ../client-go
	k8s.io/code-generator => ../code-generator
	k8s.io/component-base => ../component-base
	k8s.io/sample-apiserver => ../sample-apiserver
)
