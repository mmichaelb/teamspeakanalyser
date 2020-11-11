package teamspeakanalyser

import (
	"fmt"
	"github.com/multiplay/go-ts3"
	"github.com/neo4j/neo4j-go-driver/neo4j"
	"log"
)

type Analyser struct {
	config          *Config
	teamSpeakClient *ts3.Client
	neo4jDriver     neo4j.Driver
	neo4jSession    neo4j.Session
}

func New(config *Config) *Analyser {
	return &Analyser{config: config}
}

func (analyser *Analyser) Connect() {
	log.Println("Connecting to TeamSpeak server...")
	if err := analyser.connectTeamSpeak(); err != nil {
		log.Fatalf("Could not connect to TeamSpeak server: %e", err)
	}
	log.Println("Connected to TeamSpeak server!")
	log.Println("Connecting to Neo4j database server...")
	if err := analyser.connectNeo4j(); err != nil {
		log.Fatalf("Could not connect to Neo4j database server: %e", err)
	}
	log.Println("Connected to Neo4j database server!")
}

func (analyser *Analyser) connectTeamSpeak() (err error) {
	config := analyser.config.TeamSpeak
	serverAddress := fmt.Sprintf("%s:%d", config.Host, config.Port)
	log.Printf("Using TeamSpeak Server Address: %s", serverAddress)
	analyser.teamSpeakClient, err = ts3.NewClient(serverAddress)
	if err != nil {
		return err
	}
	if err := analyser.teamSpeakClient.Login(config.User, config.Password); err != nil {
		return err
	}
	if version, err := analyser.teamSpeakClient.Version(); err != nil {
		return err
	} else {
		log.Printf("TeamSpeak is running version: %+v\n", version)
	}
	return nil
}

func (analyser *Analyser) connectNeo4j() (err error) {
	config := analyser.config.Neo4j
	uri := fmt.Sprintf("%s:%d", config.Host, config.Port)
	analyser.neo4jDriver, err = neo4j.NewDriver(uri, neo4j.BasicAuth(config.User, config.Password, ""), func(c *neo4j.Config) {
		c.Encrypted = config.Encrypted
	})
	if err != nil {
		return
	}
	analyser.neo4jSession, err = analyser.neo4jDriver.Session(neo4j.AccessModeWrite)
	if err != nil {
		return
	}
	version, err := analyser.neo4jSession.ReadTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
		result, err := transaction.Run("CALL dbms.components() YIELD name, versions, edition UNWIND versions AS version RETURN name, version, edition;", map[string]interface{}{})
		if err != nil {
			return nil, err
		}
		if result.Next() {
			return result.Record().Values(), nil
		}
		return nil, result.Err()
	})
	if err != nil {
		return
	}
	log.Printf("Neo4j server is running version: %+v", version)
	return
}
