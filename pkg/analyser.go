package teamspeakanalyser

import (
	"github.com/multiplay/go-ts3"
	"github.com/neo4j/neo4j-go-driver/neo4j"
	"log"
	"os"
)

type Analyser struct {
	config                    *Config
	teamSpeakClient           *ts3.Client
	teamSpeakNotificationChan chan ts3.Notification
	neo4jDriver               neo4j.Driver
	neo4jSession              neo4j.Session
}

func New(config *Config) *Analyser {
	return &Analyser{config: config}
}

func (analyser *Analyser) Connect() {
	log.Println("Connecting to Neo4j database server...")
	if err := analyser.connectNeo4j(); err != nil {
		log.Printf("Could not connect to Neo4j database server: %v", err)
		analyser.Shutdown()
		return
	}
	log.Println("Connected to Neo4j database server!")
	log.Println("Connecting to TeamSpeak server...")
	if err := analyser.connectTeamSpeak(); err != nil {
		log.Printf("Could not connect to TeamSpeak server: %v", err)
		analyser.Shutdown()
		return
	}
	if err := analyser.setupTeamSpeak(); err != nil {
		log.Printf("Could not setup TeamSpeak server: %v", err)
		analyser.Shutdown()
		return
	}
	log.Println("Connected to TeamSpeak server!")
}

func (analyser *Analyser) Shutdown() {
	log.Println("Shutting down analyser...")
	analyser.closeTeamSpeak()
	analyser.closeNeo4j()
	os.Exit(0)
}
