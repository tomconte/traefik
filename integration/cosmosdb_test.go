package integration

import (
	"net/http"
	"os"
	"time"

	"github.com/containous/traefik/integration/try"
	"github.com/containous/traefik/log"
	"github.com/containous/traefik/types"
	"github.com/go-check/check"
	checker "github.com/vdemeester/shakers"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type CosmosDBSuite struct {
	BaseSuite
}

func (s *CosmosDBSuite) SetUpSuite(c *check.C) {
	s.createComposeProject(c, "cosmosdb")
	s.composeProject.Start(c)

	url := "mongodb://" + s.composeProject.Container(c, "cosmosdb").NetworkSettings.IPAddress + ":27017"
	//url := "mongodb://172.27.0.1:27017"
	log.Println("Connecting to: " + url)

	var session *mgo.Session
	err := try.Do(60*time.Second, func() error {
		s, err := mgo.Dial(url)
		if err != nil {
			log.Printf("Dial error: %s", err)
			return err
		}
		session = s
		return nil
	})
	c.Assert(err, checker.IsNil)

	// Create collection
	collection := session.DB("test").C("traefik")

	// Create some items
	whoami1 := "http://" + s.composeProject.Container(c, "whoami1").NetworkSettings.IPAddress + ":80"
	whoami2 := "http://" + s.composeProject.Container(c, "whoami2").NetworkSettings.IPAddress + ":80"
	whoami3 := "http://" + s.composeProject.Container(c, "whoami3").NetworkSettings.IPAddress + ":80"

	backend := struct {
		Name    string
		Backend types.Backend
	}{
		Name: "whoami",
		Backend: types.Backend{
			Servers: map[string]types.Server{
				"whoami1": {
					URL: whoami1,
				},
				"whoami2": {
					URL: whoami2,
				},
				"whoami3": {
					URL: whoami3,
				},
			},
		},
	}

	frontend := struct {
		Name     string
		Frontend types.Frontend
	}{
		Name: "whoami",
		Frontend: types.Frontend{
			EntryPoints: []string{
				"http",
			},
			Backend: "whoami",
			Routes: map[string]types.Route{
				"hostRule": {
					Rule: "Host:test.traefik.io",
				},
			},
		},
	}

	log.Println("Inserting documents...")

	err = collection.Insert(&backend, &frontend)

	if err != nil {
		log.Println(err)
		return
	}

	// Make sure the items have been created

	n, err := collection.Find(bson.M{"backend": bson.M{"$exists": true}}).Count()

	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("Docs inserted: %d backend", n)

	n, err = collection.Find(bson.M{"frontend": bson.M{"$exists": true}}).Count()

	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("Docs inserted: %d frontend", n)
}

func (s *CosmosDBSuite) TestCosmosDB(c *check.C) {
	ip := s.composeProject.Container(c, "cosmosdb").NetworkSettings.IPAddress
	file := s.adaptFile(c, "fixtures/cosmosdb/simple.toml", struct{ CosmosIP string }{ip})
	defer os.Remove(file)

	cmd, display := s.traefikCmd(withConfigFile(file))
	defer display(c)
	err := cmd.Start()
	c.Assert(err, checker.IsNil)
	defer cmd.Process.Kill()

	err = try.GetRequest("http://127.0.0.1:8081/api/providers", 120*time.Second, try.BodyContains("Host:test.traefik.io"))
	c.Assert(err, checker.IsNil)

	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:8080/", nil)
	c.Assert(err, checker.IsNil)
	req.Host = "test.traefik.io"
	err = try.Request(req, 200*time.Millisecond, try.StatusCodeIs(http.StatusOK))
	c.Assert(err, checker.IsNil)
}

func (s *CosmosDBSuite) TearDownSuite(c *check.C) {
	if s.composeProject != nil {
		s.composeProject.Stop(c)
	}
}
