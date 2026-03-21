package provider

import (
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
)

// Name is the Pulumi provider name used for registration.
const Name = "elasticstack"

// Version is set via ldflags at build time: -X provider/pkg/provider.Version=vX.Y.Z
var Version = "0.1.0"

// NewProvider builds the Pulumi provider with all resources and functions registered.
func NewProvider(resources []infer.InferredResource, functions []infer.InferredFunction) (p.Provider, error) {
	return infer.NewProviderBuilder().
		WithDescription("A Pulumi native provider for Elastic Stack (Elasticsearch, Kibana, Fleet, APM, Cloud).").
		WithPluginDownloadURL("github://api.github.com/dbmurphy").
		WithConfig(infer.Config(&Config{})).
		WithResources(resources...).
		WithFunctions(functions...).
		WithModuleMap(map[tokens.ModuleName]tokens.ModuleName{
			"elasticsearch/functions": "elasticsearch",
			"kibana/functions":        "kibana",
			"kibana/space":            "kibana",
			"kibana/security":         "kibana",
			"kibana/alerting":         "kibana",
			"kibana/dataview":         "kibana",
			"kibana/savedobj":         "kibana",
			"kibana/slo":              "kibana",
			"kibana/detection":        "kibana",
			"kibana/synthetics":       "kibana",
			"kibana/dashboard":        "kibana",
			"fleet/functions":         "fleet",
			"fleet":                   "fleet",
			"apm":                     "apm",
			"cloud":                   "cloud",
		}).
		Build()
}
