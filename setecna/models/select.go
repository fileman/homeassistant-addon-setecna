package models

type Select struct {
	CommandTemplate string `json:"command_template"`
	CommandTopic    string `json:"command_topic"`
	Device          struct {
		Manufacturer     string   `json:"manufacturer"`
		Model            string   `json:"model"`
		Name             string   `json:"name"`
		Identifiers      []string `json:"identifiers"`
		ConfigurationURL string   `json:"configuration_url,omitempty"`
		SwVersion        string   `json:"sw_version,omitempty"`
	} `json:"device"`
	AvailabilityTopic string   `json:"availability_topic,omitempty"`
	EntityCategory    string   `json:"entity_category"`
	Name              string   `json:"name"`
	Options           []string `json:"options"`
	StateTopic        string   `json:"state_topic"`
	UniqueID          string   `json:"unique_id"`
	ValueTemplate     string   `json:"value_template"`
}

func (s *Select) Init(systemID, sensorID string, attributes Attributes) {
	s.CommandTemplate = attributes.CommandTemplate
	s.CommandTopic = "homeassistant/" + attributes.EntityType + "/" + systemID + "_" + sensorID + "/set"
	s.Device.Manufacturer = "Setecna"
	s.Device.Model = "REG system"
	s.Device.Name = systemID
	s.Device.Identifiers = []string{systemID}
	s.Device.ConfigurationURL = "https://www.s5a.eu/station/" + systemID
	s.Device.SwVersion = SwVersion
	s.AvailabilityTopic = "setecna/" + systemID + "/status"
	s.EntityCategory = "config"
	s.Name = attributes.Name
	s.Options = attributes.Options
	s.StateTopic = "homeassistant/" + attributes.EntityType + "/" + systemID + "_" + sensorID
	s.UniqueID = systemID + "_" + sensorID
	s.ValueTemplate = attributes.ValueTemplate
}
