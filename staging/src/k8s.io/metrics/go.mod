// This is a generated file. Do not edit directly.

module k8s.io/metrics

go 1.16

require (
	github.com/gogo/protobuf v1.3.2
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/stretchr/testify v1.7.0
	google.golang.org/appengine v1.6.7 // indirect
	k8s.io/api v0.0.0
	k8s.io/apimachinery v0.0.0
	k8s.io/client-go v0.0.0
	k8s.io/code-generator v0.0.0
)

replace (
	k8s.io/api => ../api
	k8s.io/apimachinery => ../apimachinery
	k8s.io/client-go => ../client-go
	k8s.io/code-generator => ../code-generator
	k8s.io/metrics => ../metrics
)
