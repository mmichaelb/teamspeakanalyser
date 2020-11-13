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
	log.Println("Closing Neo4j server connection...")
	err := analyser.neo4jDriver.Close()
	if err != nil {
		log.Printf("Could not close Neo4j server connection: %v", err)
	} else {
		log.Println("Closed Neo4j server connection.")
	}
}

func (analyser *Analyser) setupNeo4j() error {
	if err := analyser.createNeo4jConstraints(); err != nil {
		return err
	}
	if err := analyser.createNeo4jNameIndex(); err != nil {
		return err
	}
	return nil
}

func (analyser *Analyser) createNeo4jConstraints() error {
	if err := analyser.createNeo4jUniqueUserConstraint("user_cid", "cid"); err != nil {
		return err
	}
	if err := analyser.createNeo4jUniqueUserConstraint("user_uid", "uid"); err != nil {
		return err
	}
	return nil
}

func (analyser *Analyser) createNeo4jUniqueUserConstraint(name, fieldName string) error {
	constraintsAdded, err := analyser.neo4jSession.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
		result, err := transaction.Run(fmt.Sprintf("CREATE CONSTRAINT %s IF NOT EXISTS ON (u:User) ASSERT u.%s IS UNIQUE", name, fieldName), map[string]interface{}{})
		if err != nil {
			return nil, err
		}
		summary, err := result.Summary()
		if err != nil {
			return nil, err
		}
		return summary.Counters().ConstraintsAdded(), nil
	})
	if constraintsAdded == 1 {
		log.Printf(`Created constraint "%s" for node "User".`, name)
	}
	if err != nil {
		return err
	}
	return nil
}

func (analyser *Analyser) createNeo4jNameIndex() error {
	indexAdded, err := analyser.neo4jSession.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
		result, err := transaction.Run("CREATE INDEX user_name IF NOT EXISTS FOR (u:User) ON (u.name)", map[string]interface{}{})
		if err != nil {
			return nil, err
		}
		summary, err := result.Summary()
		if err != nil {
			return nil, err
		}
		return summary.Counters().IndexesAdded(), nil
	})
	if indexAdded == 1 {
		log.Println(`Created index "user_name" for User nodes.`)
	}
	if err != nil {
		return err
	}
	return nil
}
