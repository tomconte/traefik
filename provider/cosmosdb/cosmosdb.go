package cosmosdb

import (
	"fmt"
	"time"

	"github.com/containous/traefik/log"
	"github.com/containous/traefik/provider"
	"github.com/containous/traefik/safe"
	"github.com/containous/traefik/types"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var _ provider.Provider = (*Provider)(nil)

// Provider holds configuration for provider.
type Provider struct {
	provider.BaseProvider `mapstructure:",squash" export:"true"`
	Host                  string `description:"Host"`
	Port                  int    `description:"Port"`
	Username              string `description:"Username"`
	Password              string `description:"Password"`
	Database              string `description:"Database"`
	CollectionName        string `description:"CollectionName"`
}

type backendDoc struct {
	ID      bson.ObjectId `bson:"_id,omitempty"`
	Name    string        `bson:"name,omitempty"`
	Backend types.Backend `bson:"backend,omitempty"`
}

type frontendDoc struct {
	ID       bson.ObjectId  `bson:"_id,omitempty"`
	Name     string         `bson:"name,omitempty"`
	Frontend types.Frontend `bson:"frontend,omitempty"`
}

func (p *Provider) queryCosmosData() ([]backendDoc, []frontendDoc, error) {
	dialInfo := &mgo.DialInfo{
		Addrs:    []string{fmt.Sprintf("%s:%d", p.Host, p.Port)},
		Timeout:  60 * time.Second,
		Database: p.Database,
		Username: p.Username,
		Password: p.Password,
	}

	/*
		DialServer: func(addr *mgo.ServerAddr) (net.Conn, error) {
		return tls.Dial("tcp", addr.String(), &tls.Config{})
	},*/

	log.Debugf("Connecting to %s:%d", p.Host, p.Port)

	session, err := mgo.DialWithInfo(dialInfo)

	if err != nil {
		return nil, nil, err
	}

	defer session.Close()

	collection := session.DB(p.Database).C(p.CollectionName)

	var backendDocs []backendDoc
	err = collection.Find(bson.M{"backend": bson.M{"$exists": true}}).All(&backendDocs)

	log.Debugf("Retrieved %d backend docs", len(backendDocs))

	if err != nil {
		return nil, nil, err
	}

	var frontendDocs []frontendDoc
	err = collection.Find(bson.M{"frontend": bson.M{"$exists": true}}).All(&frontendDocs)

	log.Debugf("Retrieved %d frontend docs", len(frontendDocs))

	if err != nil {
		return nil, nil, err
	}

	return backendDocs, frontendDocs, nil
}

func (p *Provider) loadCosmosConfig(backendDocs []backendDoc, frontendDocs []frontendDoc) (*types.Configuration, error) {
	backends := map[string]*types.Backend{}
	for _, v := range backendDocs {
		backends[v.Name] = &v.Backend
	}

	frontends := map[string]*types.Frontend{}
	for _, v := range frontendDocs {
		frontends[v.Name] = &v.Frontend
	}

	return &types.Configuration{
		Backends:  backends,
		Frontends: frontends,
	}, nil
}

// Provide provides the configuration to traefik via the configuration channel.
func (p *Provider) Provide(configurationChan chan<- types.ConfigMessage, pool *safe.Pool, constraints types.Constraints) error {
	log.Debugf("CosmosDB provider")

	backends, frontends, err := p.queryCosmosData()

	if err != nil {
		return err
	}

	log.Debugf("Got %d backends, %d frontends", len(backends), len(frontends))

	configuration, err := p.loadCosmosConfig(backends, frontends)

	if err != nil {
		return err
	}

	configurationChan <- types.ConfigMessage{
		ProviderName:  "cosmosdb",
		Configuration: configuration,
	}

	return nil
}
