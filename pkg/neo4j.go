package teamspeakanalyser

import (
	"fmt"
	"github.com/neo4j/neo4j-go-driver/neo4j"
	"log"
)

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
