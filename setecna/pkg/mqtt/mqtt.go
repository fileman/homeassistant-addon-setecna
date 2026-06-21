package mqtt

import (
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type Message struct {
	Topic   string `json:"topic"`
	Message string `json:"message"`
	Qos     int    `json:"qos"`
	Retain  bool   `json:"retain"`
}

type Messages []Message

type MqttServer struct {
	client MQTT.Client
}

func (s *MqttServer) Connect(host, port, user, password, availabilityTopic string) {
	opts := MQTT.NewClientOptions()
	opts.AddBroker("tcp://" + host + ":" + port)
	opts.SetUsername(user)
	opts.SetPassword(password)
	opts.SetClientID("SetecnaScraperAddon")
	// Last-Will: if the add-on dies, the broker marks the device offline so all
	// entities referencing this availability topic become "unavailable" in HA.
	// On every (re)connect we re-publish "online" so a transient reconnect does
	// not leave entities stuck on the retained Last-Will "offline".
	if availabilityTopic != "" {
		opts.SetWill(availabilityTopic, "offline", 1, true)
		opts.SetOnConnectHandler(func(c MQTT.Client) {
			c.Publish(availabilityTopic, 1, true, "online")
		})
	}

	s.client = MQTT.NewClient(opts)
	if token := s.client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
}

// PublishRetained publishes a retained QoS 1 message (used for the availability
// topic so late-subscribing entities immediately learn the online/offline state,
// and so token.Wait() blocks until the broker acknowledges — important for the
// "offline" published just before a clean shutdown/disconnect).
func (s *MqttServer) PublishRetained(topic, payload string) {
	token := s.client.Publish(topic, 1, true, payload)
	token.Wait()
}

func (s *MqttServer) Disconnect() {
	s.client.Disconnect(250)
}

func (s *MqttServer) BatchPublish(u Messages, delay int64) {
	for _, m := range u {
		// fmt.Println("---- Publishing message \"" + m.Message + "\" on topic \"" + m.Topic + "\"")
		token := s.client.Publish(m.Topic, byte(m.Qos), m.Retain, m.Message)
		token.Wait()
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
}

func (s *MqttServer) Publish(m Message) {
	// fmt.Println("---- Publishing message \"" + m.Message + "\" on topic \"" + m.Topic + "\"")
	token := s.client.Publish(m.Topic, byte(m.Qos), m.Retain, m.Message)
	token.Wait()
}

func (s *MqttServer) ListenForChange(topic string, handlerFunction MQTT.MessageHandler) {
	s.client.Subscribe(topic, 0, handlerFunction)
}
