package teamspeakanalyser

import (
	"github.com/multiplay/go-ts3"
	"github.com/neo4j/neo4j-go-driver/neo4j"
	"log"
)

type Analyser struct {
	config                *Config
	teamSpeakClient       *ts3.Client
	neo4jDriver           neo4j.Driver
	neo4jSession          neo4j.Session
	teamSpeakReadStopChan chan struct{}
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
