package teamspeakanalyser

import (
	"fmt"
	"github.com/neo4j/neo4j-go-driver/neo4j"
	"log"
)

func (analyser *Analyser) connectNeo4j() (err error) {
	config := analyser.config.Neo4j
	analyser.neo4jDriver, err = neo4j.NewDriver(config.Uri, neo4j.BasicAuth(config.User, config.Password, ""), func(c *neo4j.Config) {
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

func (analyser *Analyser) closeNeo4j() {
	if analyser.neo4jDriver == nil {
		return
	}
	fmt.Println("Closing Neo4j server connection...")
	err := analyser.neo4jDriver.Close()
	if err != nil {
		fmt.Printf("Could not close Neo4j server connection: %v", err)
	} else {
		fmt.Println("Closed Neo4j server connection.")
	}
}
