package main

import (
	"flag"
	teamspeakanalyser "github.com/mmichaelb/teamspeakanalyser/pkg"
	"log"
)

func main() {
	configFilepath := flag.String("config", "config.yml", "Specify the filepath of the config to use.")
	flag.Parse()

	config, err := teamspeakanalyser.ReadConfig(*configFilepath)
	if err != nil {
		log.Fatal(err)
	}
	analyser := teamspeakanalyser.New(config)
	analyser.Connect()
	analyser.StartListening()
}
