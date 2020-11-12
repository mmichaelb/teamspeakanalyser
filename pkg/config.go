package teamspeakanalyser

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
	"time"
)

func ReadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var config *Config
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(config)
	return nil, err
}

func WriteConfig(filename string, config *Config) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	configFileHeader := fmt.Sprintf("# Config written on %s\n", time.Now().Format("Mon Jan 2 15:04:05 -0700 MST 2006"))
	if _, err = file.Write([]byte(configFileHeader)); err != nil {
		return err
	}
	defer file.Close()
	decoder := yaml.NewEncoder(file)
	err = decoder.Encode(config)
	return err
}

type Config struct {
	TeamSpeak struct {
		Host            string `yaml:"host"`
		Port            int    `yaml:"port"`
		User            string `yaml:"user"`
		Password        string `yaml:"password"`
		VirtualServerId int    `yaml:"virtual_server_id"`
	} `yaml:"teamspeak"`
	Neo4j struct {
		Host      string `yaml:"host"`
		Port      int    `yaml:"port"`
		User      string `yaml:"user"`
		Password  string `yaml:"password"`
		Encrypted bool   `yaml:"encrypted"`
	}
}
