package main

import (
	"flag"
	teamspeakanalyser "github.com/mmichaelb/teamspeakanalyser/pkg"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"os/signal"
)

var configFilepath = flag.String("config", "config.yml", "Specify the filepath of the config to use.")
var logLevel = flag.String("loglevel", "info", "Specify the log level to use.")
var logFile = flag.String("logfile", "app.log", "Specify the log file to log to.")

func main() {
	flag.Parse()
	setupLogger()
	config, err := teamspeakanalyser.ReadConfig(*configFilepath)
	if err != nil {
		log.Fatal(err)
	}
	analyser := teamspeakanalyser.New(config)
	analyser.Connect()
	analyser.StartListening()
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt)
	<-signalChannel
	log.Println("Shutting down...")
	analyser.Shutdown()
	log.Println("Bye!")
}

func setupLogger() {
	if level, err := log.ParseLevel(*logLevel); err != nil {
		log.WithError(err).WithField("logLevel", *logLevel).Warnln("could not set custom log level - falling back to INFO")
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(level)
	}
	log.SetFormatter(&log.TextFormatter{
		ForceQuote: true,
	})
	var file *os.File
	if _, err := os.Stat(*logFile); os.IsNotExist(err) {
		file, err = os.Create(*logFile)
		if err != nil {
			log.WithField("logFile", *logFile).Fatalln("could not create log file")
		}
	} else {
		file, err = os.OpenFile(*logFile, os.O_RDWR, 0666)
		if err != nil {
			log.WithField("logFile", *logFile).Fatalln("could not open log file")
		}
	}
	log.SetOutput(io.MultiWriter(os.Stdout, file))
}
