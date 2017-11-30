package cosmosdb

import (
	"reflect"
	"testing"

	"github.com/containous/traefik/types"
)

var backend = types.Backend{
	HealthCheck: &types.HealthCheck{
		Path: "/build",
	},
	Servers: map[string]types.Server{
		"server1": {
			URL: "http://test.traefik.io",
		},
	},
}

var frontend = types.Frontend{
	EntryPoints: []string{"http"},
	Backend:     "test.traefik.io",
	Routes: map[string]types.Route{
		"route1": {
			Rule: "Host:test.traefik.io",
		},
	},
}

func TestLoadCosmosConfigSuccessful(t *testing.T) {
	provider := Provider{}

	backendDocs := make([]backendDoc, 1)
	frontendDocs := make([]frontendDoc, 1)

	backendDocs[0] = backendDoc{
		ID:      "0",
		Name:    "backend0",
		Backend: backend,
	}

	frontendDocs[0] = frontendDoc{
		ID:       "0",
		Name:     "frontend0",
		Frontend: frontend,
	}

	loadedConfig, err := provider.loadCosmosConfig(backendDocs, frontendDocs)

	if err != nil {
		t.Fatal(err)
	}

	expectedConfig := &types.Configuration{
		Backends: map[string]*types.Backend{
			"backend0": &backend,
		},
		Frontends: map[string]*types.Frontend{
			"frontend0": &frontend,
		},
	}

	if !reflect.DeepEqual(loadedConfig, expectedConfig) {
		t.Fatalf("Configurations did not match: %v %v", loadedConfig, expectedConfig)
	}
}
