package provider

import (
	"testing"
)

func TestResolveElasticsearchConfig_Defaults(t *testing.T) {
	c := &Config{}
	es := c.resolveElasticsearchConfig()
	if es == nil {
		t.Fatal("expected non-nil ES config")
	}
}

func TestResolveElasticsearchConfig_Explicit(t *testing.T) {
	c := &Config{
		Elasticsearch: &ElasticsearchConfig{
			Endpoints: []string{"https://es:9200"},
			Username:  "elastic",
			Password:  "pass",
		},
	}
	es := c.resolveElasticsearchConfig()
	if len(es.Endpoints) != 1 || es.Endpoints[0] != "https://es:9200" {
		t.Fatalf("expected https://es:9200, got %v", es.Endpoints)
	}
	if es.Username != "elastic" {
		t.Fatalf("expected 'elastic', got %s", es.Username)
	}
}

func TestResolveKibanaConfig_InheritsFromES(t *testing.T) {
	c := &Config{
		Elasticsearch: &ElasticsearchConfig{
			Username: "elastic",
			Password: "espass",
			APIKey:   "eskey",
		},
	}
	kb := c.resolveKibanaConfig()
	if kb.Username != "elastic" {
		t.Fatalf("expected kibana username to inherit 'elastic', got %s", kb.Username)
	}
	if kb.Password != "espass" {
		t.Fatalf("expected kibana password to inherit 'espass', got %s", kb.Password)
	}
	if kb.APIKey != "eskey" {
		t.Fatalf("expected kibana apiKey to inherit 'eskey', got %s", kb.APIKey)
	}
}

func TestResolveKibanaConfig_ExplicitOverridesES(t *testing.T) {
	c := &Config{
		Elasticsearch: &ElasticsearchConfig{
			Username: "elastic",
			Password: "espass",
		},
		Kibana: &KibanaConfig{
			Username: "kibana_user",
			Password: "kibana_pass",
		},
	}
	kb := c.resolveKibanaConfig()
	if kb.Username != "kibana_user" {
		t.Fatalf("expected 'kibana_user', got %s", kb.Username)
	}
	if kb.Password != "kibana_pass" {
		t.Fatalf("expected 'kibana_pass', got %s", kb.Password)
	}
}

func TestResolveFleetConfig_InheritsFromKibana(t *testing.T) {
	c := &Config{
		Elasticsearch: &ElasticsearchConfig{
			Username: "elastic",
			Password: "espass",
		},
		Kibana: &KibanaConfig{
			Username: "kibana_user",
		},
	}
	fl := c.resolveFleetConfig()
	// Fleet inherits from Kibana, which has explicit username but inherits password from ES
	if fl.Username != "kibana_user" {
		t.Fatalf("expected 'kibana_user', got %s", fl.Username)
	}
	if fl.Password != "espass" {
		t.Fatalf("expected fleet password to inherit 'espass' via kibana, got %s", fl.Password)
	}
}

func TestResolveFleetConfig_ExplicitOverrides(t *testing.T) {
	c := &Config{
		Elasticsearch: &ElasticsearchConfig{
			Username: "elastic",
			Password: "espass",
		},
		Fleet: &FleetConfig{
			Username: "fleet_user",
			Password: "fleet_pass",
		},
	}
	fl := c.resolveFleetConfig()
	if fl.Username != "fleet_user" {
		t.Fatalf("expected 'fleet_user', got %s", fl.Username)
	}
	if fl.Password != "fleet_pass" {
		t.Fatalf("expected 'fleet_pass', got %s", fl.Password)
	}
}

func TestESClient_NilWhenNotConfigured(t *testing.T) {
	c := &Config{}
	_, err := c.ESClient()
	if err == nil {
		t.Fatal("expected error when ES client not configured")
	}
}

func TestKibanaClient_NilWhenNotConfigured(t *testing.T) {
	c := &Config{}
	_, err := c.KibanaClient()
	if err == nil {
		t.Fatal("expected error when Kibana client not configured")
	}
}

func TestFleetClient_NilWhenNotConfigured(t *testing.T) {
	c := &Config{}
	_, err := c.FleetClient()
	if err == nil {
		t.Fatal("expected error when Fleet client not configured")
	}
}
