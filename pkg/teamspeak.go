package teamspeakanalyser

import (
	"fmt"
	"github.com/multiplay/go-ts3"
	"log"
)

const clIdKeyName string = "clid"

func (analyser *Analyser) connectTeamSpeak() (err error) {
	config := analyser.config.TeamSpeak
	serverAddress := fmt.Sprintf("%s:%d", config.Host, config.Port)
	log.Printf("Using TeamSpeak Server Address: %s", serverAddress)
	analyser.teamSpeakClient, err = ts3.NewClient(serverAddress)
	if err != nil {
		return err
	}
	if err := analyser.teamSpeakClient.Login(config.User, config.Password); err != nil {
		return err
	}
	if version, err := analyser.teamSpeakClient.Version(); err != nil {
		return err
	} else {
		log.Printf("TeamSpeak is running version: %+v\n", version)
	}
	return
}

func (analyser *Analyser) setupTeamSpeak() (err error) {
	config := analyser.config.TeamSpeak
	if err := analyser.teamSpeakClient.Use(config.VirtualServerId); err != nil {
		return err
	}
	if err := analyser.teamSpeakClient.Register(ts3.ServerEvents); err != nil {
		return err
	}
	if err := analyser.teamSpeakClient.Register(ts3.ChannelEvents); err != nil {
		return err
	}

	notificationChannel := make(chan ts3.Notification, ts3.DefaultNotifyBufSize)
	go func() {
	infiniteLoop:
		for {
			select {
			case notification := <-analyser.teamSpeakClient.Notifications():
				clId, ok := notification.Data[clIdKeyName]
				notificationChannel <- notification
				analyser.checkForSecondTeamSpeakNotification(notification, ok, clId, notificationChannel)
				break
			case <-analyser.teamSpeakReadStopChan:
				break infiniteLoop
			}
		}
		log.Println("Closing TeamSpeak server connection...")
		if err := analyser.teamSpeakClient.Close(); err != nil {
			log.Printf("Could not close TeamSpeak server connection: %e", err)
		} else {
			log.Println("Closed TeamSpeak server connection.")
		}
	}()
	return nil
}

func (analyser *Analyser) checkForSecondTeamSpeakNotification(notification ts3.Notification, ok bool, clId string, notificationChannel chan ts3.Notification) {
	secondNotification := <-analyser.teamSpeakClient.Notifications()
	secondClientUniqueIdentifier, secondOk := secondNotification.Data[clIdKeyName]
	if notification.Type != secondNotification.Type || !ok || !secondOk || clId != secondClientUniqueIdentifier {
		notificationChannel <- secondNotification
	}
}