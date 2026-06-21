package models

type Number struct {
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
	AvailabilityTopic string  `json:"availability_topic,omitempty"`
	DeviceClass       string  `json:"device_class,omitempty"`
	EntityCategory    string  `json:"entity_category"`
	Min               float64 `json:"min"`
	Max               float64 `json:"max"`
	Mode              string  `json:"mode"`
	Name              string  `json:"name"`
	StateTopic        string  `json:"state_topic"`
	Step              float64 `json:"step"`
	UniqueID          string  `json:"unique_id"`
	UnitOfMeasurement string  `json:"unit_of_measurement"`
	ValueTemplate     string  `json:"value_template"`
}

func (n *Number) Init(systemID, sensorID string, attributes Attributes) {
	n.CommandTemplate = attributes.CommandTemplate
	n.CommandTopic = "homeassistant/" + attributes.EntityType + "/" + systemID + "_" + sensorID + "/set"
	n.Device.Manufacturer = "Setecna"
	n.Device.Model = "REG system"
	n.Device.Name = systemID
	n.Device.Identifiers = []string{systemID}
	n.Device.ConfigurationURL = "https://www.s5a.eu/station/" + systemID
	n.Device.SwVersion = SwVersion
	n.AvailabilityTopic = "setecna/" + systemID + "/status"
	n.DeviceClass = attributes.DeviceClass
	n.EntityCategory = "config"
	n.Min = attributes.Min
	n.Max = attributes.Max
	n.Mode = "slider"
	n.Name = attributes.Name
	n.StateTopic = "homeassistant/" + attributes.EntityType + "/" + systemID + "_" + sensorID
	n.Step = attributes.Step
	n.UniqueID = systemID + "_" + sensorID
	n.UnitOfMeasurement = attributes.UnitOfMeasurement
	n.ValueTemplate = attributes.ValueTemplate
}
