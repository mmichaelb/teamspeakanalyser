package teamspeakanalyser

import (
	"github.com/multiplay/go-ts3"
	"github.com/neo4j/neo4j-go-driver/neo4j"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

type Analyser struct {
	config          *Config
	teamSpeakClient *ts3.Client
	neo4jDriver     neo4j.Driver
	neo4jSession    neo4j.Session
	closeChan       chan struct{}
	interval        time.Duration
	omitChannels    []int
}

func New(config *Config) *Analyser {
	return &Analyser{
		config:    config,
		closeChan: make(chan struct{}, 1),
	}
}

func (analyser *Analyser) Connect() {
	log.Println("connecting to Neo4j database server...")
	if err := analyser.connectNeo4j(); err != nil {
		log.WithError(err).Errorln("could not connect to Neo4j database server")
		analyser.Shutdown()
		return
	}
	if err := analyser.setupNeo4j(); err != nil {
		log.WithError(err).Errorln("could not setup Neo4j")
		analyser.Shutdown()
		return
	}
	log.Println("connected to Neo4j database server")
	log.Println("connecting to TeamSpeak server...")
	if err := analyser.connectTeamSpeak(); err != nil {
		log.WithError(err).Errorln("could not connect to TeamSpeak server")
		analyser.Shutdown()
		return
	}
	if err := analyser.setupTeamSpeak(); err != nil {
		log.WithError(err).Errorln("could not setup TeamSpeak server")
		analyser.Shutdown()
		return
	}
	log.Println("connected to TeamSpeak server")
	var err error
	analyser.interval, err = time.ParseDuration(analyser.config.Query.Interval)
	if err != nil {
		log.WithError(err).Errorln("could not parse interval duration from config")
		analyser.Shutdown()
		return
	}
}

func (analyser *Analyser) Shutdown() {
	log.Println("shutting down analyser...")
	analyser.closeChan <- struct{}{}
	// wait for stop
	<-analyser.closeChan
	analyser.closeTeamSpeak()
	analyser.closeNeo4j()
	os.Exit(0)
}
