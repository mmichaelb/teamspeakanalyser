package main

import teamspeakanalyser "github.com/mmichaelb/teamspeakanalyser/pkg"

func main() {
	defaultConfig := &teamspeakanalyser.Config{
		TeamSpeak: struct {
			Host            string `yaml:"host"`
			Port            int    `yaml:"port"`
			User            string `yaml:"user"`
			Password        string `yaml:"password"`
			VirtualServerId int    `yaml:"virtual_server_id"`
		}{Host: "127.0.0.1", Port: 10011, User: "admin", Password: "admin", VirtualServerId: 1},
		Neo4j: struct {
			Host      string `yaml:"host"`
			Port      int    `yaml:"port"`
			User      string `yaml:"user"`
			Password  string `yaml:"password"`
			Encrypted bool   `yaml:"encrypted"`
		}{Host: "127.0.0.1", Port: 7474, User: "admin", Password: "admin", Encrypted: false},
	}
	if err := teamspeakanalyser.WriteConfig("./configs/default.yml", defaultConfig); err != nil {
		panic(err)
	}
}
