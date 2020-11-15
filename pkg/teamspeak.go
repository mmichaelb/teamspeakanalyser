package teamspeakanalyser

import (
	"fmt"
	"github.com/multiplay/go-ts3"
	"github.com/neo4j/neo4j-go-driver/neo4j"
	"log"
	"strconv"
	"time"
)

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

func (analyser *Analyser) closeTeamSpeak() {
	if analyser.teamSpeakClient == nil {
		return
	}
	log.Println("Closing TeamSpeak server connection...")
	err := analyser.teamSpeakClient.Close()
	if err != nil {
		log.Printf("Could not close TeamSpeak server connection: %v", err)
	} else {
		log.Println("Closed TeamSpeak server connection.")
	}
}

func (analyser *Analyser) setupTeamSpeak() (err error) {
	config := analyser.config.TeamSpeak
	if err := analyser.teamSpeakClient.Use(config.VirtualServerId); err != nil {
		return err
	}
	analyser.omitChannels = []int{}
	channels, err := analyser.teamSpeakClient.Server.ChannelList()
	if err != nil {
		return err
	}
	for _, channel := range channels {
		for _, omitChannel := range analyser.config.Query.OmitChannels {
			if omitChannel == channel.ChannelName {
				analyser.omitChannels = append(analyser.omitChannels, channel.ID)
			}
		}
	}
	return nil
}

type clientInfo struct {
	ID                 int    `ms:"clid"`
	UniqueIdentifier   string `ms:"client_unique_identifier"`
	ChannelID          int    `ms:"cid"`
	DatabaseID         int    `ms:"client_database_id"`
	Nickname           string `ms:"client_nickname"`
	Type               int    `ms:"client_type"`
	Away               bool   `ms:"client_away"`
	AwayMessage        string `ms:"client_away_message"`
	FlagTalking        bool   `ms:"client_flag_talking"`
	InputMuted         bool   `ms:"client_input_muted"`
	OutputMuted        bool   `ms:"client_output_muted"`
	InputHardware      bool   `ms:"client_input_hardware"`
	OutputHardware     bool   `ms:"client_output_hardware"`
	TalkPower          int    `ms:"client_talk_power"`
	IsTalker           bool   `ms:"client_is_talker"`
	IsPrioritySpeaker  int    `ms:"client_is_priority_speaker"`
	IsRecording        int    `ms:"client_is_recording"`
	IsChannelCommander int    `ms:"client_is_channel_commander"`
}

func (analyser *Analyser) StartListening() {
listenerLoop:
	for {
		select {
		case closeStruct := <-analyser.closeChan:
			analyser.closeChan <- closeStruct
			break listenerLoop
		case <-time.After(analyser.interval):
			analyser.updateDatabase()
		}
	}
}

func (analyser *Analyser) updateDatabase() bool {
	log.Println("Updating database...")
	var clientList []*clientInfo
	_, err := analyser.teamSpeakClient.ExecCmd(ts3.NewCmd("clientlist").WithOptions("-voice", "-uid").WithResponse(&clientList))
	if err != nil {
		log.Printf("Could not retrieve clientInfo list: %v", err)
		return false
	}
	for _, clientInfo := range clientList {
		if clientInfo.Type != 0 {
			// omit non-standard client connections (e.g. Query-Connections)
			continue
		}
		if created, err := analyser.createNeo4jUserEntry(clientInfo); err != nil {
			log.Printf("Could not create new node entry for %s (%s): %v", strconv.Quote(clientInfo.Nickname), clientInfo.UniqueIdentifier, err)
			return false
		} else if created {
			log.Printf("Created new node entry for user %s (%s)", strconv.Quote(clientInfo.Nickname), clientInfo.UniqueIdentifier)
		}
	}
	channels, err := analyser.teamSpeakClient.Server.ChannelList()
	if err != nil {
		log.Printf("Could not retrieve channel list: %v", err)
		return false
	}
	for _, channel := range channels {
		// ignore unimportant channels
		if channel.TotalClients == 0 {
			continue
		}
		if created, err := analyser.createNeo4jChannelEntry(channel); err != nil {
			log.Printf("Could not create new node entry channel %s (%d): %v", strconv.Quote(channel.ChannelName), channel.ID, err)
			return false
		} else if created {
			log.Printf("Created new node entry for channel %s (%d).", strconv.Quote(channel.ChannelName), channel.ID)
		}
	}
	channelClientMapping := analyser.mapClients(clientList)
	for channelId, clients := range channelClientMapping {
		for _, clientInfo := range clients {
			weightName := determineIncrementWeight(clientInfo)
			if err := analyser.registerChannelInteraction(channelId, clientInfo, weightName); err != nil {
				log.Printf("Could not update channel interaction for channel %d and user %s (%s): %v", channelId, strconv.Quote(clientInfo.Nickname), clientInfo.UniqueIdentifier, err)
				return false
			}
			if len(clients) == 1 {
				// add suffix if user is alone in the channel
				weightName = fmt.Sprintf("%s_alone", weightName)
			}
			if err := analyser.registerSelfInteraction(clientInfo, weightName); err != nil {
				log.Printf("Could not update self interaction for %s (%s): %v", strconv.Quote(clientInfo.Nickname), clientInfo.UniqueIdentifier, err)
				return false
			}
			for _, clientTalkTo := range clients {
				if clientTalkTo.UniqueIdentifier == clientInfo.UniqueIdentifier {
					continue
				}
				if err := analyser.registerOtherInteraction(clientInfo, clientTalkTo, weightName); err != nil {
					log.Printf("Could not update interaction from %s (%s) to %s (%s): %v", strconv.Quote(clientInfo.Nickname), clientInfo.UniqueIdentifier, strconv.Quote(clientTalkTo.Nickname), clientTalkTo.UniqueIdentifier, err)
					return false
				}
			}
		}
	}
	return true
}

func (analyser *Analyser) isChannelOmitted(clientInfo *clientInfo) bool {
	for _, channelId := range analyser.omitChannels {
		if channelId == clientInfo.ChannelID {
			return true
		}
	}
	return false
}

func (analyser *Analyser) mapClients(clientList []*clientInfo) map[int][]*clientInfo {
	channelClientMapping := make(map[int][]*clientInfo)
	for _, client := range clientList {
		if client.Type != 0 || analyser.isChannelOmitted(client) {
			// omit non-standard client connections (e.g. Query-Connections) and ignored channels
			continue
		}
		channelClientList, ok := channelClientMapping[client.ChannelID]
		if ok {
			channelClientList = append(channelClientList, client)
		} else {
			channelClientList = []*clientInfo{client}
		}
		channelClientMapping[client.ChannelID] = channelClientList
	}
	return channelClientMapping
}

func (analyser *Analyser) registerSelfInteraction(clientInfo *clientInfo, weightName string) error {
	return analyser.registerUserInteraction(clientInfo, clientInfo, weightName)
}

func (analyser *Analyser) registerOtherInteraction(clientInfo *clientInfo, talkToClientInfo *clientInfo, weightName string) error {
	return analyser.registerUserInteraction(clientInfo, talkToClientInfo, weightName)
}

func (analyser *Analyser) registerUserInteraction(clientInfo *clientInfo, talkToClientInfo *clientInfo, weightName string) error {
	_, err := analyser.neo4jSession.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
		query := fmt.Sprintf("MATCH (u:User),(u2:User) WHERE u.uid = $uid AND u2.uid = $talkToUid "+
			"MERGE (u)-[h:HANGS_WITH]->(u2) "+
			"WITH h, COALESCE(h.%s, 0) as old_count "+
			"SET h.last_interaction = datetime(), h.%s = old_count + $amount", weightName, weightName)
		result, err := transaction.Run(query, map[string]interface{}{
			"uid":       clientInfo.UniqueIdentifier,
			"talkToUid": talkToClientInfo.UniqueIdentifier,
			"amount":    int64(analyser.interval.Seconds()),
		})
		if err != nil {
			return nil, err
		}
		if result.Err() != nil {
			return nil, result.Err()
		}
		return nil, nil
	})
	return err
}

func (analyser Analyser) registerChannelInteraction(channelId int, clientInfo *clientInfo, weightName string) error {
	_, err := analyser.neo4jSession.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
		query := fmt.Sprintf("MATCH (c:Channel),(u:User) WHERE c.id = $id AND u.uid = $uid "+
			"MERGE (u)-[h:HANGS_IN]->(c) "+
			"WITH h, COALESCE(h.%s, 0) as old_count "+
			"SET h.last_interaction = datetime(), h.%s = old_count + $amount", weightName, weightName)
		result, err := transaction.Run(query, map[string]interface{}{
			"id":     channelId,
			"uid":    clientInfo.UniqueIdentifier,
			"amount": int64(analyser.interval.Seconds()),
		})
		if err != nil {
			return nil, err
		}
		if result.Err() != nil {
			return nil, result.Err()
		}
		return nil, nil
	})
	return err
}

func determineIncrementWeight(clientInfo *clientInfo) string {
	if !clientInfo.OutputHardware || clientInfo.OutputMuted {
		return "wt_output"
	} else if !clientInfo.InputHardware || clientInfo.InputMuted {
		return "wt_input"
	} else if clientInfo.Away {
		return "wt_away"
	}
	return "wt_unmuted"
}

func (analyser *Analyser) createNeo4jUserEntry(clientInfo *clientInfo) (bool, error) {
	created, err := analyser.neo4jSession.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
		result, err := transaction.Run("MERGE (u:User {uid:$uid}) "+
			"ON CREATE SET u.name = $name, u.clid = $clid "+
			"ON MATCH SET u.name = $name", map[string]interface{}{
			"uid":  clientInfo.UniqueIdentifier,
			"name": clientInfo.Nickname,
			"clid": clientInfo.ID,
		})
		if err != nil {
			return false, err
		}
		if result.Err() != nil {
			return nil, result.Err()
		}
		summary, err := result.Summary()
		if err != nil {
			return false, err
		}
		return summary.Counters().NodesCreated() >= 1, nil
	})
	if created == nil {
		return false, err
	}
	return created.(bool), nil
}

func (analyser *Analyser) createNeo4jChannelEntry(channel *ts3.Channel) (bool, error) {
	created, err := analyser.neo4jSession.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
		result, err := transaction.Run("MERGE (c:Channel {id:$id}) "+
			"SET u.name = $name", map[string]interface{}{
			"id":   channel.ID,
			"name": channel.ChannelName,
		})
		if err != nil {
			return false, err
		}
		if result.Err() != nil {
			return nil, result.Err()
		}
		summary, err := result.Summary()
		if err != nil {
			return false, err
		}
		return summary.Counters().NodesCreated() >= 1, nil
	})
	if created == nil {
		return false, err
	}
	return created.(bool), nil
}
