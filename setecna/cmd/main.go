package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Ingordigia/homeassistant-addon-setecna/models"
	"github.com/Ingordigia/homeassistant-addon-setecna/pkg/config"
	"github.com/Ingordigia/homeassistant-addon-setecna/pkg/helpers"
	"github.com/Ingordigia/homeassistant-addon-setecna/pkg/homeassistant"
	"github.com/Ingordigia/homeassistant-addon-setecna/pkg/mqtt"
	"github.com/Ingordigia/homeassistant-addon-setecna/pkg/scraper"
)

var systemID string = os.Getenv("REG_SYSTEM_ID")
var username string = os.Getenv("REG_USER")
var password string = os.Getenv("REG_PASSWORD")

var mqttHost string = os.Getenv("MQTT_HOST")
var mqttPort string = os.Getenv("MQTT_PORT")
var mqttUser string = os.Getenv("MQTT_USER")
var mqttPassword string = os.Getenv("MQTT_PASSWORD")

var advInt, isReadonly, debugDump bool

func main() {

	var err error
	advInt, err = helpers.GetenvBool("ADV_INT")
	if err != nil {
		log.Println("adv_int parameter non specified, default is false")
		advInt = false
	}
	isReadonly, err = helpers.GetenvBool("READONLY")
	if err != nil {
		log.Println("readonly parameter non specified, default is true")
		isReadonly = true
	}
	debugDump, err = helpers.GetenvBool("DEBUG_DUMP")
	if err != nil {
		debugDump = false
	}

	if mqttPort == "" {
		mqttPort = "1883"
	}

	availabilityTopic := "setecna/" + systemID + "/status"

	mqttServer := new(mqtt.MqttServer)
	// Connect publishes "online" to availabilityTopic via its OnConnect handler.
	mqttServer.Connect(mqttHost, mqttPort, mqttUser, mqttPassword, availabilityTopic)

	scraper := new(scraper.Scraper)
	scraper.Init(systemID, debugDump)

	// CONNECT TO SETECNA SERVERS
	for {
		err = scraper.Login(username, password)
		if err != nil {
			log.Print("Unable to login to Setecna servers: ", err)
			log.Print("Try again in 30 seconds")
			time.Sleep(time.Second * 30)
		} else {
			break
		}
	}
	log.Print("Connected to remote Setercna servers")

	go do(mqttServer, scraper)

	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	<-quitChannel
	//time for cleanup before exit
	mqttServer.PublishRetained(availabilityTopic, "offline")
	mqttServer.Disconnect()
	log.Println("Service stopped")
}

func do(m *mqtt.MqttServer, s *scraper.Scraper) {

	var message []byte
	var fetchError, err error = nil, nil
	var sensor = new(models.Sensor)
	var number = new(models.Number)
	var selection = new(models.Select)
	var binarySensor = new(models.BinarySensor)
	var response scraper.Response

	// FETCH DATA FROM SETECNA SERVERS
	for {
		response, fetchError = s.Fetch()
		if fetchError != nil {
			log.Print("Unable to fetch data from Setecna servers: ", err)
			log.Print("Try again in 30 seconds")
			time.Sleep(time.Second * 30)
		} else {
			break
		}
	}
	log.Print("Start creating entities in Home Assistant")

	responseMap := make(map[string]string)
	for _, num := range response.Data {
		responseMap[num.ID] = string(num.V)
	}

	// DEBUG: dump the parameters returned by the station that the add-on does
	// not (yet) map to an entity, so new parameters can be discovered/mapped.
	if debugDump {
		known := make(models.ParamsMap)
		known.AddEnabledParams(responseMap, isReadonly)
		known.AddDisabledParams(responseMap, isReadonly)
		unmapped := []map[string]string{}
		for _, num := range response.Data {
			if _, ok := known[num.ID]; !ok {
				unmapped = append(unmapped, map[string]string{"Id": num.ID, "V": string(num.V)})
			}
		}
		if derr := config.Save("/share/setecna_unmapped_ids.json", unmapped); derr != nil {
			log.Println("[debug] unmapped-ids dump failed:", derr)
		} else {
			log.Printf("[debug] wrote %d unmapped param ids to /share/setecna_unmapped_ids.json", len(unmapped))
		}
	}

	// MANAGE HA BUILT-IN ENTITIES
	if advInt && !isReadonly {
		// CREATE CLIMATE
		configMessages := homeassistant.CreateClimates(responseMap, systemID)
		m.BatchPublish(configMessages, 100)
	} else {
		// REMOVE CLIMATE
		configMessages := homeassistant.RemoveClimates(responseMap, systemID)
		m.BatchPublish(configMessages, 100)
	}

	// REMOVE UNDESIRED ENTITIES FROM HA
	toRemoveParams := make(models.ParamsMap)
	toRemoveParams.AddDisabledParams(responseMap, isReadonly)
	for paramKey, paramAttributes := range toRemoveParams {
		removeMessage := mqtt.Message{
			Topic:   "homeassistant/" + paramAttributes.EntityType + "/" + systemID + "_" + paramKey + "/config",
			Message: "",
			Qos:     0,
			Retain:  true,
		}
		m.Publish(removeMessage)
		time.Sleep(time.Millisecond * 100)
	}

	// CREATE OR UPDATE HA ENTITIES
	params := make(models.ParamsMap)
	params.AddEnabledParams(responseMap, isReadonly)
	for paramKey, paramAttributes := range params {
		// if _, ok := responseMap[paramKey]; ok { // Removed because LAST_UPDATE is not part of responseMap, maby I should add it and re-enable this check
		switch paramAttributes.EntityType {
		case "sensor":
			sensor.Init(systemID, paramKey, paramAttributes)
			message, err = json.Marshal(sensor)
			if err != nil {
				log.Println(err)
				return
			}
		case "number":
			number.Init(systemID, paramKey, paramAttributes)
			message, err = json.Marshal(number)
			if err != nil {
				log.Println(err)
				return
			}
		case "select":
			selection.Init(systemID, paramKey, paramAttributes)
			message, err = json.Marshal(selection)
			if err != nil {
				log.Println(err)
				return
			}
		case "binary_sensor":
			binarySensor.Init(systemID, paramKey, paramAttributes)
			message, err = json.Marshal(binarySensor)
			if err != nil {
				log.Println(err)
				return
			}
		}
		configMessage := mqtt.Message{
			Topic:   "homeassistant/" + paramAttributes.EntityType + "/" + systemID + "_" + paramKey + "/config",
			Message: string(message),
			Qos:     0,
			Retain:  true,
		}
		m.Publish(configMessage)
		time.Sleep(time.Millisecond * 100)
		// }
	}

	// START LISTENING FOR CHANGE
	if !isReadonly {
		go m.ListenForChange("homeassistant/+/+/set", s.Change(systemID))
	}

	// WAITING FOR HA TO CREATE ALL ENTITIES
	time.Sleep(time.Second * 1)

	log.Print("Entities created, starting main loop of data synchronization with Home Assistant")

	// START MAIN LOOP
	for {
		if fetchError == nil {
			// log.Println("------------------")
			// log.Println(response)
			messages := response.GetUpdatedValues(systemID, params)
			m.BatchPublish(messages, 100)
		} else {
			log.Print("Unable to fetch data from Setecna servers: ", err)
			log.Print("Try again in 30 seconds")
		}
		time.Sleep(time.Second * 10)
		s.AskRefresh()
		time.Sleep(time.Second * 20)
		response, fetchError = s.Fetch()
	}
}
