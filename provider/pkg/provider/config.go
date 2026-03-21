// Package provider implements the Pulumi provider configuration.
package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/clients"
)

// Config holds the provider configuration for all connection blocks.
type Config struct {
	Elasticsearch *ElasticsearchConfig `pulumi:"elasticsearch,optional"`
	Kibana        *KibanaConfig        `pulumi:"kibana,optional"`
	Fleet         *FleetConfig         `pulumi:"fleet,optional"`
	Cloud         *CloudConfig         `pulumi:"cloud,optional"`

	// Global provider settings
	DestroyProtection bool `pulumi:"destroyProtection,optional"`

	// Initialized clients (not serialized)
	esClient     *clients.ElasticsearchClient
	kibanaClient *clients.KibanaClient
	fleetClient  *clients.FleetClient
	cloudClient  *clients.CloudClient
}

// CloudConfig holds Elastic Cloud API connection configuration.
type CloudConfig struct {
	Endpoint string `pulumi:"endpoint,optional"`
	APIKey   string `pulumi:"apiKey,optional"   provider:"secret"`
}

// ElasticsearchConfig holds Elasticsearch connection configuration.
type ElasticsearchConfig struct {
	Endpoints              []string          `pulumi:"endpoints,optional"`
	Username               string            `pulumi:"username,optional"`
	Password               string            `pulumi:"password,optional"               provider:"secret"`
	APIKey                 string            `pulumi:"apiKey,optional"                 provider:"secret"`
	BearerToken            string            `pulumi:"bearerToken,optional"            provider:"secret"`
	ESClientAuthentication string            `pulumi:"esClientAuthentication,optional" provider:"secret"`
	Insecure               bool              `pulumi:"insecure,optional"`
	CAFile                 string            `pulumi:"caFile,optional"`
	CAData                 string            `pulumi:"caData,optional"`
	CertFile               string            `pulumi:"certFile,optional"`
	CertData               string            `pulumi:"certData,optional"`
	KeyFile                string            `pulumi:"keyFile,optional"`
	KeyData                string            `pulumi:"keyData,optional"                provider:"secret"`
	Headers                map[string]string `pulumi:"headers,optional"                provider:"secret"`
}

// KibanaConfig holds Kibana connection configuration.
type KibanaConfig struct {
	Endpoints   []string `pulumi:"endpoints,optional"`
	Username    string   `pulumi:"username,optional"`
	Password    string   `pulumi:"password,optional"    provider:"secret"`
	APIKey      string   `pulumi:"apiKey,optional"      provider:"secret"`
	BearerToken string   `pulumi:"bearerToken,optional" provider:"secret"`
	CACerts     []string `pulumi:"caCerts,optional"`
	Insecure    bool     `pulumi:"insecure,optional"`
}

// FleetConfig holds Fleet connection configuration.
type FleetConfig struct {
	Endpoint    string   `pulumi:"endpoint,optional"`
	Username    string   `pulumi:"username,optional"`
	Password    string   `pulumi:"password,optional"    provider:"secret"`
	APIKey      string   `pulumi:"apiKey,optional"      provider:"secret"`
	BearerToken string   `pulumi:"bearerToken,optional" provider:"secret"`
	CACerts     []string `pulumi:"caCerts,optional"`
	Insecure    bool     `pulumi:"insecure,optional"`
}

// Annotate sets provider configuration property descriptions and defaults.
func (c *Config) Annotate(a infer.Annotator) {
	a.Describe(&c.Elasticsearch, "Elasticsearch connection configuration.")
	a.Describe(&c.Kibana, "Kibana connection configuration.")
	a.Describe(&c.Fleet, "Fleet connection configuration.")
	a.Describe(&c.Cloud, "Elastic Cloud API connection configuration.")
	a.Describe(
		&c.DestroyProtection,
		"When true, no resource is deleted on destroy — the entire stack becomes abandon-on-destroy mode.",
	)
	a.SetDefault(&c.DestroyProtection, false)
}

// Annotate sets Cloud configuration property descriptions and defaults.
func (c *CloudConfig) Annotate(a infer.Annotator) {
	a.Describe(&c.Endpoint, "Elastic Cloud API endpoint URL.")
	a.SetDefault(&c.Endpoint, "https://api.elastic-cloud.com", "EC_ENDPOINT")
	a.Describe(&c.APIKey, "Elastic Cloud API key for authentication.")
	a.SetDefault(&c.APIKey, nil, "EC_API_KEY")
}

// Annotate sets Elasticsearch configuration property descriptions and defaults.
func (c *ElasticsearchConfig) Annotate(a infer.Annotator) {
	a.Describe(&c.Endpoints, "List of Elasticsearch endpoint URLs. Falls back to ELASTICSEARCH_ENDPOINTS env var.")
	a.Describe(&c.Username, "Username for basic authentication.")
	a.SetDefault(&c.Username, nil, "ELASTICSEARCH_USERNAME")
	a.Describe(&c.Password, "Password for basic authentication.")
	a.SetDefault(&c.Password, nil, "ELASTICSEARCH_PASSWORD")
	a.Describe(&c.APIKey, "API key for authentication (base64 encoded).")
	a.SetDefault(&c.APIKey, nil, "ELASTICSEARCH_API_KEY")
	a.Describe(&c.BearerToken, "Bearer token for JWT authentication.")
	a.SetDefault(&c.BearerToken, nil, "ELASTICSEARCH_BEARER_TOKEN")
	a.Describe(&c.ESClientAuthentication, "Shared secret for ES-Client JWT authentication.")
	a.SetDefault(&c.ESClientAuthentication, nil, "ELASTICSEARCH_ES_CLIENT_AUTHENTICATION")
	a.Describe(&c.Insecure, "Skip TLS certificate verification.")
	a.SetDefault(&c.Insecure, false, "ELASTICSEARCH_INSECURE")
	a.Describe(&c.CAFile, "Path to a CA certificate file for TLS verification.")
	a.Describe(&c.CAData, "PEM-encoded CA certificate data for TLS verification.")
	a.Describe(&c.CertFile, "Path to a client certificate file for mTLS.")
	a.Describe(&c.CertData, "PEM-encoded client certificate data for mTLS.")
	a.Describe(&c.KeyFile, "Path to a client private key file for mTLS.")
	a.Describe(&c.KeyData, "PEM-encoded client private key data for mTLS.")
	a.Describe(&c.Headers, "Custom HTTP headers to send with every request.")
}

// Annotate sets Kibana configuration property descriptions and defaults.
func (c *KibanaConfig) Annotate(a infer.Annotator) {
	a.Describe(&c.Endpoints, "List of Kibana endpoint URLs. Falls back to KIBANA_ENDPOINT env var.")
	a.Describe(&c.Username, "Username for basic authentication.")
	a.SetDefault(&c.Username, nil, "KIBANA_USERNAME")
	a.Describe(&c.Password, "Password for basic authentication.")
	a.SetDefault(&c.Password, nil, "KIBANA_PASSWORD")
	a.Describe(&c.APIKey, "API key for authentication.")
	a.SetDefault(&c.APIKey, nil, "KIBANA_API_KEY")
	a.Describe(&c.BearerToken, "Bearer token for JWT authentication.")
	a.SetDefault(&c.BearerToken, nil, "KIBANA_BEARER_TOKEN")
	a.Describe(&c.CACerts, "List of CA certificate file paths for TLS verification. Falls back to KIBANA_CA_CERTS env var.")
	a.Describe(&c.Insecure, "Skip TLS certificate verification.")
	a.SetDefault(&c.Insecure, false, "KIBANA_INSECURE")
}

// Annotate sets Fleet configuration property descriptions and defaults.
func (c *FleetConfig) Annotate(a infer.Annotator) {
	a.Describe(&c.Endpoint, "Fleet/Kibana endpoint URL.")
	a.SetDefault(&c.Endpoint, nil, "FLEET_ENDPOINT")
	a.Describe(&c.Username, "Username for basic authentication.")
	a.SetDefault(&c.Username, nil, "FLEET_USERNAME")
	a.Describe(&c.Password, "Password for basic authentication.")
	a.SetDefault(&c.Password, nil, "FLEET_PASSWORD")
	a.Describe(&c.APIKey, "API key for authentication.")
	a.SetDefault(&c.APIKey, nil, "FLEET_API_KEY")
	a.Describe(&c.BearerToken, "Bearer token for JWT authentication.")
	a.SetDefault(&c.BearerToken, nil, "FLEET_BEARER_TOKEN")
	a.Describe(&c.CACerts, "List of CA certificate file paths for TLS verification. Falls back to FLEET_CA_CERTS env var.")
	a.Describe(&c.Insecure, "Skip TLS certificate verification.")
}

// Configure validates the provider configuration and initializes HTTP clients.
// This is called on every Pulumi operation — credentials are resolved fresh each time.
func (c *Config) Configure(_ context.Context) error {
	esConfig := c.resolveElasticsearchConfig()
	if esConfig != nil && len(esConfig.Endpoints) > 0 {
		client, err := clients.NewElasticsearchClient(esConfig.toClientConfig())
		if err != nil {
			return fmt.Errorf("failed to create Elasticsearch client: %w", err)
		}
		c.esClient = client
	}

	kibanaConfig := c.resolveKibanaConfig()
	if kibanaConfig != nil && len(kibanaConfig.Endpoints) > 0 {
		client, err := clients.NewKibanaClient(kibanaConfig.toKibanaClientConfig())
		if err != nil {
			return fmt.Errorf("failed to create Kibana client: %w", err)
		}
		c.kibanaClient = client
	}

	fleetConfig := c.resolveFleetConfig()
	if fleetConfig != nil && fleetConfig.Endpoint != "" {
		client, err := clients.NewFleetClient(fleetConfig.toFleetClientConfig())
		if err != nil {
			return fmt.Errorf("failed to create Fleet client: %w", err)
		}
		c.fleetClient = client
	}

	cloudConfig := c.resolveCloudConfig()
	if cloudConfig != nil && cloudConfig.APIKey != "" {
		client, err := clients.NewCloudClient(clients.CloudClientConfig{
			Endpoint: cloudConfig.Endpoint,
			APIKey:   cloudConfig.APIKey,
		})
		if err != nil {
			return fmt.Errorf("failed to create Cloud client: %w", err)
		}
		c.cloudClient = client
	}

	return nil
}

// ESClient returns the Elasticsearch client, or an error if not configured.
func (c *Config) ESClient() (*clients.ElasticsearchClient, error) {
	if c.esClient == nil {
		return nil, fmt.Errorf(
			"elasticsearch is not configured; set endpoints via provider config or ELASTICSEARCH_ENDPOINTS env var",
		)
	}
	return c.esClient, nil
}

// KibanaClient returns the Kibana client, or an error if not configured.
func (c *Config) KibanaClient() (*clients.KibanaClient, error) {
	if c.kibanaClient == nil {
		return nil, fmt.Errorf("kibana is not configured; set endpoints via provider config or KIBANA_ENDPOINT env var")
	}
	return c.kibanaClient, nil
}

// FleetClient returns the Fleet client, or an error if not configured.
func (c *Config) FleetClient() (*clients.FleetClient, error) {
	if c.fleetClient == nil {
		return nil, fmt.Errorf("fleet is not configured; set endpoint via provider config or FLEET_ENDPOINT env var")
	}
	return c.fleetClient, nil
}

// CloudClient returns the Elastic Cloud client, or an error if not configured.
func (c *Config) CloudClient() (*clients.CloudClient, error) {
	if c.cloudClient == nil {
		return nil, fmt.Errorf("elastic cloud is not configured; set apiKey via provider config or EC_API_KEY env var")
	}
	return c.cloudClient, nil
}

// resolveCloudConfig returns the Cloud config, falling back to env vars.
func (c *Config) resolveCloudConfig() *CloudConfig {
	if c.Cloud != nil {
		return c.Cloud
	}
	cc := &CloudConfig{}
	if v := os.Getenv("EC_ENDPOINT"); v != "" {
		cc.Endpoint = v
	} else {
		cc.Endpoint = "https://api.elastic-cloud.com"
	}
	if v := os.Getenv("EC_API_KEY"); v != "" {
		cc.APIKey = v
	}
	return cc
}

// splitEndpoints parses an endpoints env var value. Supports both
// JSON array format (["http://a","http://b"]) and comma-separated.
func splitEndpoints(v string) []string {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, "[") {
		// JSON array: strip brackets and quotes
		v = strings.Trim(v, "[]")
		var out []string
		for _, s := range strings.Split(v, ",") {
			s = strings.TrimSpace(s)
			s = strings.Trim(s, `"'`)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return strings.Split(v, ",")
}

// resolveElasticsearchConfig returns the effective ES config.
// When the user omits the elasticsearch block entirely, the infer framework's
// applyDefaults skips nil optional pointer-to-struct fields, so SetDefault env
// var bindings from Annotate never fire. We read env vars manually here.
func (c *Config) resolveElasticsearchConfig() *ElasticsearchConfig {
	if c.Elasticsearch != nil {
		return c.Elasticsearch
	}
	es := &ElasticsearchConfig{}
	if v := os.Getenv("ELASTICSEARCH_ENDPOINTS"); v != "" {
		es.Endpoints = splitEndpoints(v)
	}
	if v := os.Getenv("ELASTICSEARCH_USERNAME"); v != "" {
		es.Username = v
	}
	if v := os.Getenv("ELASTICSEARCH_PASSWORD"); v != "" {
		es.Password = v
	}
	if v := os.Getenv("ELASTICSEARCH_API_KEY"); v != "" {
		es.APIKey = v
	}
	if v := os.Getenv("ELASTICSEARCH_BEARER_TOKEN"); v != "" {
		es.BearerToken = v
	}
	if v := os.Getenv("ELASTICSEARCH_ES_CLIENT_AUTHENTICATION"); v != "" {
		es.ESClientAuthentication = v
	}
	if v := os.Getenv("ELASTICSEARCH_INSECURE"); v != "" {
		es.Insecure, _ = strconv.ParseBool(v)
	}
	return es
}

// resolveKibanaConfig implements credential inheritance: Kibana falls back to ES credentials.
func (c *Config) resolveKibanaConfig() *KibanaConfig {
	if c.Kibana == nil {
		c.Kibana = &KibanaConfig{}
	}
	kb := c.Kibana
	es := c.resolveElasticsearchConfig()

	if kb.Username == "" && es != nil {
		kb.Username = es.Username
	}
	if kb.Password == "" && es != nil {
		kb.Password = es.Password
	}
	if kb.APIKey == "" && es != nil {
		kb.APIKey = es.APIKey
	}
	if kb.BearerToken == "" && es != nil {
		kb.BearerToken = es.BearerToken
	}

	return kb
}

// resolveFleetConfig implements credential inheritance: Fleet falls back to Kibana → ES.
func (c *Config) resolveFleetConfig() *FleetConfig {
	if c.Fleet == nil {
		c.Fleet = &FleetConfig{}
	}
	fl := c.Fleet
	kb := c.resolveKibanaConfig()

	if fl.Username == "" && kb != nil {
		fl.Username = kb.Username
	}
	if fl.Password == "" && kb != nil {
		fl.Password = kb.Password
	}
	if fl.APIKey == "" && kb != nil {
		fl.APIKey = kb.APIKey
	}
	if fl.BearerToken == "" && kb != nil {
		fl.BearerToken = kb.BearerToken
	}

	return fl
}

// toClientConfig converts ElasticsearchConfig to the clients package config type.
func (c *ElasticsearchConfig) toClientConfig() clients.ElasticsearchClientConfig {
	return clients.ElasticsearchClientConfig{
		Endpoints:              c.Endpoints,
		Username:               c.Username,
		Password:               c.Password,
		APIKey:                 c.APIKey,
		BearerToken:            c.BearerToken,
		ESClientAuthentication: c.ESClientAuthentication,
		Insecure:               c.Insecure,
		CAFile:                 c.CAFile,
		CAData:                 c.CAData,
		CertFile:               c.CertFile,
		CertData:               c.CertData,
		KeyFile:                c.KeyFile,
		KeyData:                c.KeyData,
		Headers:                c.Headers,
	}
}

// toKibanaClientConfig converts KibanaConfig to the clients package config type.
func (c *KibanaConfig) toKibanaClientConfig() clients.KibanaClientConfig {
	return clients.KibanaClientConfig{
		Endpoints:   c.Endpoints,
		Username:    c.Username,
		Password:    c.Password,
		APIKey:      c.APIKey,
		BearerToken: c.BearerToken,
		CACerts:     c.CACerts,
		Insecure:    c.Insecure,
	}
}

// toFleetClientConfig converts FleetConfig to the clients package config type.
func (c *FleetConfig) toFleetClientConfig() clients.FleetClientConfig {
	return clients.FleetClientConfig{
		Endpoint:    c.Endpoint,
		Username:    c.Username,
		Password:    c.Password,
		APIKey:      c.APIKey,
		BearerToken: c.BearerToken,
		CACerts:     c.CACerts,
		Insecure:    c.Insecure,
	}
}
