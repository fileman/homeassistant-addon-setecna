package models

import (
	"fmt"
	"strconv"
	"strings"
)

// SwVersion is the station firmware/release string shown in the Home Assistant
// device block. It is set once at startup (from the getres FIRMWARE_RELEASE /
// DOT_RELEASE params) before entities are built; empty means "unknown" and the
// device block omits sw_version.
var SwVersion string

// resolveLabel decodes a *_DESCR descriptor into the user's free-text label.
// The station's description fields are indices into a table: values >= 240 point
// at the free-description array (_FREEDESC1.._FREEDESC16), with 240 -> _FREEDESC1.
// Values below 240 index a built-in (untranslated) table and 0 means "no label".
// Returns "" when there is no usable free-text label. This holds across every
// family (zones, circuits, sources, analog inputs, ...) — _DESCR is always a
// name index, never a type code.
func resolveLabel(from map[string]string, descrKey string) string {
	d, err := strconv.Atoi(strings.TrimSpace(from[descrKey]))
	if err != nil || d < 240 {
		return ""
	}
	// 240..255 -> _FREEDESC1.._FREEDESC16; 256+ continues into the extended
	// _XFREEDESC table (offset inferred — confirm on a station that uses more
	// than 16 free descriptions). Out-of-range lookups return "" and the caller
	// falls back to the generic name.
	if d >= 256 {
		return strings.TrimSpace(from["_XFREEDESC"+strconv.Itoa(d-255)])
	}
	return strings.TrimSpace(from["_FREEDESC"+strconv.Itoa(d-239)])
}

// LabelOr returns the decoded free-text label for descrKey, or fallback when the
// station has no custom label for it.
func LabelOr(from map[string]string, descrKey, fallback string) string {
	if l := resolveLabel(from, descrKey); l != "" {
		return l
	}
	return fallback
}

type Attributes struct {
	CommandTemplate   string   `json:"command_template"`
	DeviceClass       string   `json:"device_class"`
	EntityType        string   `json:"entity_type"`
	EntityCategory    string   `json:"entity_category"`
	Max               float64  `json:"max"`
	Min               float64  `json:"min"`
	Name              string   `json:"name"`
	Options           []string `json:"options"`
	StateClass        string   `json:"state_class"`
	Step              float64  `json:"step"`
	UnitOfMeasurement string   `json:"unit_of_measurement"`
	ValueTemplate     string   `json:"value_template"`
}

type ParamsMap map[string]Attributes

func (m ParamsMap) AddEnabledParams(from map[string]string, isReadOnly bool) {
	m.addLastUpdate(from, true, isReadOnly, !isReadOnly)
	m.addGlobals(from, true, isReadOnly, !isReadOnly)
	m.addDomesticHotWater(from, true, isReadOnly, !isReadOnly)
	m.addAnalogInput(from, true, isReadOnly, !isReadOnly)
	m.addDigitalInput(from, true, isReadOnly, !isReadOnly)
	m.addDigitalAlarm(from, true, isReadOnly, !isReadOnly)
	m.addZones(from, true, isReadOnly, !isReadOnly)
	m.addCircuits(from, true, isReadOnly, !isReadOnly)
	m.addSources(from, true, isReadOnly, !isReadOnly)
	m.addDehumidifier(from, true, isReadOnly, !isReadOnly)
	m.addEnergymeters(from, true, isReadOnly, !isReadOnly)
	m.addCalendars(from, true, isReadOnly, !isReadOnly)
	m.addAlarms(from, true, isReadOnly, !isReadOnly)
	m.addSolar(from, true, isReadOnly, !isReadOnly)
	m.addDevice(from, true, isReadOnly, !isReadOnly)
	m.markDiagnostics()
}

func (m ParamsMap) AddDisabledParams(from map[string]string, isReadOnly bool) {
	m.addGlobals(from, false, !isReadOnly, isReadOnly)
	m.addDomesticHotWater(from, false, !isReadOnly, isReadOnly)
	m.addAnalogInput(from, false, !isReadOnly, isReadOnly)
	m.addDigitalInput(from, false, !isReadOnly, isReadOnly)
	m.addDigitalAlarm(from, false, !isReadOnly, isReadOnly)
	m.addZones(from, false, !isReadOnly, isReadOnly)
	m.addCircuits(from, false, !isReadOnly, isReadOnly)
	m.addSources(from, false, !isReadOnly, isReadOnly)
	m.addDehumidifier(from, false, !isReadOnly, isReadOnly)
	m.addEnergymeters(from, false, !isReadOnly, isReadOnly)
	m.addCalendars(from, false, !isReadOnly, isReadOnly)
	m.addAlarms(from, false, !isReadOnly, isReadOnly)
	m.addSolar(from, false, !isReadOnly, isReadOnly)
	m.addDevice(from, false, !isReadOnly, isReadOnly)
	m.markDiagnostics()
}

// markDiagnostics tags non-primary readouts (alarms, generic digital inputs,
// tuning/hysteresis values, configured fan flow rates and the last-update
// timestamp) as diagnostic entities. Primary measurements (temperatures,
// humidity, power, energy, operational states) are left with no entity_category
// so Home Assistant treats them as the device's primary entities. Number and
// select entities set their own category and ignore this field.
func (m ParamsMap) markDiagnostics() {
	for id, attr := range m {
		if isDiagnostic(id) {
			attr.EntityCategory = "diagnostic"
			m[id] = attr
		}
	}
}

func isDiagnostic(id string) bool {
	switch {
	case id == "LAST_UPDATE":
		return true
	case strings.HasPrefix(id, "FDIN"):
		return true
	case strings.HasPrefix(id, "FALDIN"):
		return true
	case strings.Contains(id, "_HYST"):
		return true
	case strings.Contains(id, "_DEICE_TRESH"):
		return true
	case id == "ACS_SET_DELTA":
		return true
	case strings.Contains(id, "_SPEED_"):
		return true
	default:
		return false
	}
}

func (m ParamsMap) addLastUpdate(from map[string]string, static, read, write bool) {
	if static {
		m["LAST_UPDATE"] = Attributes{
			DeviceClass: "timestamp",
			EntityType:  "sensor",
			Name:        "Last update",
		}
	}
}

func (m ParamsMap) addGlobals(from map[string]string, static, read, write bool) {
	if static {
		m["GLOBAL_ENABLE"] = Attributes{
			Name:          "Global state",
			EntityType:    "binary_sensor",
			ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
		}
		m["GLOBAL_T_EXT"] = Attributes{
			Name:              "Global external temperature",
			EntityType:        "sensor",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
		}
		m["GLOBAL_SEASON"] = Attributes{
			Name:          "Global season",
			EntityType:    "sensor",
			DeviceClass:   "enum",
			ValueTemplate: "{% if value == \"0\" %}winter{% elif value == \"1\" %}summer{% else %}{{ value }}{% endif %}",
		}
		m["GLOBAL_DEICING"] = Attributes{
			Name:          "Global de-ice state",
			EntityType:    "binary_sensor",
			ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
		}
		m["GLOBAL_EXPECTED_DEWP"] = Attributes{
			Name:              "Global dewpoint",
			EntityType:        "sensor",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
		}
	}
	if read {
		m["GLOBAL_ZONE_T_HYST"] = Attributes{
			Name:              "Global zone temperature hysteresis",
			EntityType:        "sensor",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
		}
		m["GLOBAL_ZONE_RH_HYST"] = Attributes{
			Name:              "Global zone humidity hysteresis",
			EntityType:        "sensor",
			DeviceClass:       "humidity",
			UnitOfMeasurement: "%",
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
		}
		m["GLOBAL_ZONE_DEICE_TRESH"] = Attributes{
			Name:              "Global zone de-ice threshold",
			EntityType:        "sensor",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
		}
	}
	if write {
		m["GLOBAL_ZONE_T_HYST"] = Attributes{
			Name:              "Global zone temperature hysteresis",
			EntityType:        "number",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			Max:               1,
			Min:               0.1,
			Step:              0.1,
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
			CommandTemplate:   "{{ (value * 10) | int }}",
		}
		m["GLOBAL_ZONE_RH_HYST"] = Attributes{
			Name:              "Global zone humidity hysteresis",
			EntityType:        "number",
			DeviceClass:       "humidity",
			UnitOfMeasurement: "%",
			Max:               5,
			Min:               1,
			Step:              0.1,
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
			CommandTemplate:   "{{ (value * 10) | int }}",
		}
		m["GLOBAL_ZONE_DEICE_TRESH"] = Attributes{
			Name:              "Global zone de-ice threshold",
			EntityType:        "number",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			Max:               10,
			Min:               6,
			Step:              0.1,
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
			CommandTemplate:   "{{ (value * 10) | int }}",
		}
	}
}

func (m ParamsMap) addDomesticHotWater(from map[string]string, static, read, write bool) {
	if static {
		m["ACS_MAIN_OUTPUT"] = Attributes{
			Name:          "DHW state",
			EntityType:    "binary_sensor",
			ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
		}
		m["GLOBAL_ACS_ENABLE"] = Attributes{
			Name:          "DHW enabled",
			EntityType:    "binary_sensor",
			ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
		}
		m["GLOBAL_T_ACS"] = Attributes{
			Name:              "DHW temperature",
			EntityType:        "sensor",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
		}
		m["GLOBAL_SET_ACS"] = Attributes{
			Name:              "DHW temperature setpoint",
			EntityType:        "sensor",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
		}
	}
	if read {
		m["ACS_SET_ECONOMY"] = Attributes{
			Name:              "DHW economy setpoint",
			EntityType:        "sensor",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
		}
		m["ACS_SET_COMFORT"] = Attributes{
			Name:              "DHW comfort setpoint",
			EntityType:        "sensor",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
		}
		m["ACS_SET_HYST"] = Attributes{
			Name:              "DHW setpoint hysteresis",
			EntityType:        "sensor",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
		}
		m["ACS_SET_DELTA"] = Attributes{
			Name:              "DHW second stage deviation",
			EntityType:        "sensor",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
		}
	}
	if write {
		m["ACS_SET_ECONOMY"] = Attributes{
			Name:              "DHW economy setpoint",
			EntityType:        "number",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			Max:               60,
			Min:               30,
			Step:              0.1,
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
			CommandTemplate:   "{{ (value * 10) | int }}",
		}
		m["ACS_SET_COMFORT"] = Attributes{
			Name:              "DHW comfort setpoint",
			EntityType:        "number",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			Max:               60,
			Min:               30,
			Step:              0.1,
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
			CommandTemplate:   "{{ (value * 10) | int }}",
		}
		m["ACS_SET_HYST"] = Attributes{
			Name:              "DHW setpoint hysteresis",
			EntityType:        "number",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			Max:               10,
			Min:               0,
			Step:              0.1,
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
			CommandTemplate:   "{{ (value * 10) | int }}",
		}
		m["ACS_SET_DELTA"] = Attributes{
			Name:              "DHW second stage deviation",
			EntityType:        "number",
			DeviceClass:       "temperature",
			UnitOfMeasurement: "°C",
			Max:               10,
			Min:               0,
			Step:              0.1,
			StateClass:        "measurement",
			ValueTemplate:     "{{ value | int / 10 }}",
			CommandTemplate:   "{{ (value * 10) | int }}",
		}
	}
}

func (m ParamsMap) addAnalogInput(from map[string]string, static, read, write bool) {
	if static {
		for i := 1; i <= 8; i++ {
			if from["FAIN"+fmt.Sprint(i)+"_TEMP"] != "32769" {
				// FAIN{n}_DESCR is a name index (not a type code): all analog
				// inputs read temperature; the descriptor only names them.
				m["FAIN"+fmt.Sprint(i)+"_TEMP"] = Attributes{
					Name:              LabelOr(from, "FAIN"+fmt.Sprint(i)+"_DESCR", "Analog input "+fmt.Sprint(i)),
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					StateClass:        "measurement",
					UnitOfMeasurement: "°C",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "",
				}
			}
		}
	}
}

func (m ParamsMap) addDigitalInput(from map[string]string, static, read, write bool) {
	if static {
		for i := 1; i <= 8; i++ {
			m["FDIN"+fmt.Sprint(i)+"_STATUS"] = Attributes{
				Name:          "Digital input " + fmt.Sprint(i),
				EntityType:    "binary_sensor",
				ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
			}
		}
	}
}

func (m ParamsMap) addDigitalAlarm(from map[string]string, static, read, write bool) {
	if static {
		for i := 1; i <= 5; i++ {
			m["FALDIN"+fmt.Sprint(i)+"_STATUS"] = Attributes{
				Name:          "Alarm " + fmt.Sprint(i),
				EntityType:    "binary_sensor",
				ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
			}
		}
	}
}

func (m ParamsMap) addZones(from map[string]string, static, read, write bool) {
	for i := 1; i <= 32; i++ {
		if from["Z"+fmt.Sprint(i)+"_SENSOR_CHN"] != "0" {
			base := LabelOr(from, "Z"+fmt.Sprint(i)+"_DESCR", "Zone "+fmt.Sprint(i))
			if static {
				m["Z"+fmt.Sprint(i)+"_OUTPUT"] = Attributes{
					Name:          base + " state",
					EntityType:    "binary_sensor",
					ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
				}
				m["Z"+fmt.Sprint(i)+"_TEMP"] = Attributes{
					Name:              base + " temperature",
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "{{ (value * 10) | int }}",
				}
				m["Z"+fmt.Sprint(i)+"_ZONE_MODE"] = Attributes{
					Name:          base + " mode",
					EntityType:    "sensor",
					DeviceClass:   "enum",
					ValueTemplate: "{% if value == \"0\" %}off{% elif value == \"2\" %}economy{% elif value == \"3\" %}comfort{% elif value == \"4\" %}forced off{% elif value == \"6\" %}forced economy{% elif value == \"23\" %}forced comfort{% else %}{{ value }}{% endif %}",
				}
				m["Z"+fmt.Sprint(i)+"_ZONE_SET"] = Attributes{
					Name:              base + " setpoint",
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "{{ (value * 10) | int }}",
				}
				m["Z"+fmt.Sprint(i)+"_SENSOR_CHN"] = Attributes{
					Name:           base + " sensor channel",
					EntityType:     "sensor",
					EntityCategory: "diagnostic",
					ValueTemplate:  "{{ value | int }}",
				}
				m["Z"+fmt.Sprint(i)+"_DEUM"] = Attributes{
					Name:          base + " dehumidify demand",
					EntityType:    "binary_sensor",
					ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
				}
				m["Z"+fmt.Sprint(i)+"_TEMP_OFFSET"] = Attributes{
					Name:              base + " temperature offset",
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					EntityCategory:    "diagnostic",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
				}
				m["Z"+fmt.Sprint(i)+"_LINKED"] = Attributes{
					Name:           base + " linked zone",
					EntityType:     "sensor",
					EntityCategory: "diagnostic",
					ValueTemplate:  "{{ value | int }}",
				}
				if from["Z"+fmt.Sprint(i)+"_DEWPOINT"] != "32769" {
					m["Z"+fmt.Sprint(i)+"_DEWPOINT"] = Attributes{
						Name:              base + " dewpoint",
						EntityType:        "sensor",
						DeviceClass:       "temperature",
						UnitOfMeasurement: "°C",
						StateClass:        "measurement",
						ValueTemplate:     "{{ value | int / 10 }}",
					}
				}
			}
			if read {
				m["Z"+fmt.Sprint(i)+"_FORCING"] = Attributes{
					Name:          base + " preset",
					EntityType:    "sensor",
					DeviceClass:   "enum",
					ValueTemplate: "{% if value == \"0\" %}automatic{% elif value == \"1\" %}forced off{% elif value == \"2\" %}forced economy{% elif value == \"3\" %}forced comfort{% else %}{{ value }}{% endif %}",
				}
				m["Z"+fmt.Sprint(i)+"_SET_CW"] = Attributes{
					Name:              base + " C.W. setpoint",
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "{{ (value * 10) | int }}",
				}
				m["Z"+fmt.Sprint(i)+"_SET_EW"] = Attributes{
					Name:              base + " E.W. setpoint",
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "{{ (value * 10) | int }}",
				}
				m["Z"+fmt.Sprint(i)+"_SET_CS"] = Attributes{
					Name:              base + " C.S. setpoint",
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "{{ (value * 10) | int }}",
				}
				m["Z"+fmt.Sprint(i)+"_SET_ES"] = Attributes{
					Name:              base + " E.S. setpoint",
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "{{ (value * 10) | int }}",
				}
			}
			if write {
				m["Z"+fmt.Sprint(i)+"_FORCING"] = Attributes{
					Name:            base + " preset",
					Options:         []string{"automatic", "forced off", "forced economy", "forced comfort"},
					EntityType:      "select",
					ValueTemplate:   "{% if value == \"1\" %}forced off{% elif value == \"2\" %}forced economy{% elif value == \"3\" %}forced comfort{% else %}automatic{% endif %}",
					CommandTemplate: "{% if value == \"forced off\" %}1{% elif value == \"forced economy\" %}2{% elif value == \"forced comfort\" %}3{% else %}0{% endif %}",
				}
				m["Z"+fmt.Sprint(i)+"_SET_CW"] = Attributes{
					Name:              base + " C.W. setpoint",
					EntityType:        "number",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					Max:               30,
					Min:               15,
					Step:              0.1,
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "{{ (value * 10) | int }}",
				}
				m["Z"+fmt.Sprint(i)+"_SET_EW"] = Attributes{
					Name:              base + " E.W. setpoint",
					EntityType:        "number",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					Max:               30,
					Min:               15,
					Step:              0.1,
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "{{ (value * 10) | int }}",
				}
				m["Z"+fmt.Sprint(i)+"_SET_CS"] = Attributes{
					Name:              base + " C.S. setpoint",
					EntityType:        "number",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					Max:               30,
					Min:               15,
					Step:              0.1,
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "{{ (value * 10) | int }}",
				}
				m["Z"+fmt.Sprint(i)+"_SET_ES"] = Attributes{
					Name:              base + " E.S. setpoint",
					EntityType:        "number",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					Max:               30,
					Min:               15,
					Step:              0.1,
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "{{ (value * 10) | int }}",
				}
			}
			if from["Z"+fmt.Sprint(i)+"_RH"] != "32769" {
				if static {
					m["Z"+fmt.Sprint(i)+"_RH"] = Attributes{
						Name:              base + " humidity",
						EntityType:        "sensor",
						DeviceClass:       "humidity",
						UnitOfMeasurement: "%",
						StateClass:        "measurement",
						ValueTemplate:     "{{ value | int / 10 }}",
						CommandTemplate:   "{{ (value * 10) | int }}",
					}
				}
				if read {
					m["Z"+fmt.Sprint(i)+"_SET_RH"] = Attributes{
						Name:              base + " humidity setpoint",
						EntityType:        "sensor",
						DeviceClass:       "humidity",
						UnitOfMeasurement: "%",
						StateClass:        "measurement",
						ValueTemplate:     "{{ value | int / 10 }}",
						CommandTemplate:   "{{ (value * 10) | int }}",
					}
				}
				if write {
					m["Z"+fmt.Sprint(i)+"_SET_RH"] = Attributes{
						Name:              base + " humidity setpoint",
						EntityType:        "number",
						DeviceClass:       "humidity",
						UnitOfMeasurement: "%",
						Max:               70,
						Min:               40,
						Step:              0.1,
						StateClass:        "measurement",
						ValueTemplate:     "{{ value | int / 10 }}",
						CommandTemplate:   "{{ (value * 10) | int }}",
					}
				}
			}
		}
	}
}

func (m ParamsMap) addCircuits(from map[string]string, static, read, write bool) {
	for i := 1; i <= 8; i++ {
		hasTemp := from["C"+fmt.Sprint(i)+"_TEMP"] != "32769"
		descr := from["C"+fmt.Sprint(i)+"_DESCR"]
		hasDescr := descr != "0" && descr != ""
		// A circuit exists if it has a temperature probe OR a configured
		// descriptor (some circuits drive a pump/valve without a temp probe).
		if !hasTemp && !hasDescr {
			continue
		}
		base := LabelOr(from, "C"+fmt.Sprint(i)+"_DESCR", "Circuit "+fmt.Sprint(i))
		if static {
			m["C"+fmt.Sprint(i)+"_OUTPUT"] = Attributes{
				Name:          base + " pump",
				EntityType:    "binary_sensor",
				ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
			}
			m["C"+fmt.Sprint(i)+"_MODE"] = Attributes{
				Name:           base + " mode",
				EntityType:     "sensor",
				EntityCategory: "diagnostic",
				ValueTemplate:  "{{ value | int }}",
			}
			if hasTemp {
				m["C"+fmt.Sprint(i)+"_TEMP"] = Attributes{
					Name:              base + " temperature",
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "{{ (value * 10) | int }}",
				}
				m["C"+fmt.Sprint(i)+"_SET"] = Attributes{
					Name:              base + " temperature setpoint",
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "{{ (value * 10) | int }}",
				}
			}
		}
	}
}

func (m ParamsMap) addSources(from map[string]string, static, read, write bool) {
	for i := 1; i <= 3; i++ {
		descr := from["S"+fmt.Sprint(i)+"_DESCR"]
		if descr != "0" && descr != "" {
			base := LabelOr(from, "S"+fmt.Sprint(i)+"_DESCR", "Source "+fmt.Sprint(i))
			if static {
				m["S"+fmt.Sprint(i)+"_ENABLED"] = Attributes{
					Name:          base + " enabled",
					EntityType:    "binary_sensor",
					ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
				}
				m["S"+fmt.Sprint(i)+"_OUTPUT"] = Attributes{
					Name:          base + " state",
					EntityType:    "binary_sensor",
					ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
				}
				m["S"+fmt.Sprint(i)+"_AUXOUTPUT"] = Attributes{
					Name:          base + " auxiliary state",
					EntityType:    "binary_sensor",
					ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
				}
				m["S"+fmt.Sprint(i)+"_PRIORITY"] = Attributes{
					Name:           base + " priority",
					EntityType:     "sensor",
					EntityCategory: "diagnostic",
					ValueTemplate:  "{{ value | int }}",
				}
				// 0-10V modulation output (raw word; scaling not yet confirmed).
				m["S"+fmt.Sprint(i)+"_OUTPUT_010"] = Attributes{
					Name:           base + " modulation",
					EntityType:     "sensor",
					EntityCategory: "diagnostic",
					ValueTemplate:  "{{ value | int }}",
				}
				if from["S"+fmt.Sprint(i)+"_TEMP"] != "32769" {
					m["S"+fmt.Sprint(i)+"_TEMP"] = Attributes{
						Name:              base + " temperature",
						EntityType:        "sensor",
						DeviceClass:       "temperature",
						UnitOfMeasurement: "°C",
						StateClass:        "measurement",
						ValueTemplate:     "{{ value | int / 10 }}",
					}
				}
				if from["S"+fmt.Sprint(i)+"_AUXTEMP"] != "32769" {
					m["S"+fmt.Sprint(i)+"_AUXTEMP"] = Attributes{
						Name:              base + " auxiliary temperature",
						EntityType:        "sensor",
						DeviceClass:       "temperature",
						UnitOfMeasurement: "°C",
						StateClass:        "measurement",
						ValueTemplate:     "{{ value | int / 10 }}",
					}
				}
			}
		}
	}
}

func (m ParamsMap) addDehumidifier(from map[string]string, static, read, write bool) {
	for i := 1; i <= 8; i++ {
		if from["D"+fmt.Sprint(i)+"_SPEED_LOW"] != "0" && from["D"+fmt.Sprint(i)+"_SPEED_ECONOMY"] != "0" {
			if static {
				m["D"+fmt.Sprint(i)+"_OUTPUT_RENEW"] = Attributes{
					Name:          "Fan " + fmt.Sprint(i) + " renew",
					EntityType:    "binary_sensor",
					ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
				}
				m["D"+fmt.Sprint(i)+"_OUTPUT_DEUM"] = Attributes{
					Name:          "Fan " + fmt.Sprint(i) + " dehumidify",
					EntityType:    "binary_sensor",
					ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
				}

			}
			if read {
				m["D"+fmt.Sprint(i)+"_SPEED_LOW"] = Attributes{
					Name:              "Fan " + fmt.Sprint(i) + " low flow rate",
					EntityType:        "sensor",
					UnitOfMeasurement: "m³/h",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int * 10 }}",
				}
				m["D"+fmt.Sprint(i)+"_SPEED_MED"] = Attributes{
					Name:              "Fan " + fmt.Sprint(i) + " medium flow rate",
					EntityType:        "sensor",
					UnitOfMeasurement: "m³/h",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int * 10 }}",
				}
				m["D"+fmt.Sprint(i)+"_SPEED_HIGH"] = Attributes{
					Name:              "Fan " + fmt.Sprint(i) + " high flow rate",
					EntityType:        "sensor",
					UnitOfMeasurement: "m³/h",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int * 10 }}",
				}
				m["D"+fmt.Sprint(i)+"_SPEED_BOOST"] = Attributes{
					Name:              "Fan " + fmt.Sprint(i) + " boost flow rate",
					EntityType:        "sensor",
					UnitOfMeasurement: "m³/h",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int * 10 }}",
				}
				m["D"+fmt.Sprint(i)+"_SPEED_ECONOMY"] = Attributes{
					Name:              "Fan " + fmt.Sprint(i) + " economy flow rate",
					EntityType:        "sensor",
					UnitOfMeasurement: "m³/h",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int * 10 }}",
				}
				m["D"+fmt.Sprint(i)+"_SPEED_COMFORT"] = Attributes{
					Name:              "Fan " + fmt.Sprint(i) + " comfort flow rate",
					EntityType:        "sensor",
					UnitOfMeasurement: "m³/h",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int * 10 }}",
				}
			}
			if write {
				m["D"+fmt.Sprint(i)+"_SPEED_LOW"] = Attributes{
					Name:              "Fan " + fmt.Sprint(i) + " low flow rate",
					EntityType:        "number",
					UnitOfMeasurement: "m³/h",
					Max:               250,
					Min:               100,
					Step:              10,
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int * 10 }}",
					CommandTemplate:   "{{ (value / 10) | int }}",
				}
				m["D"+fmt.Sprint(i)+"_SPEED_MED"] = Attributes{
					Name:              "Fan " + fmt.Sprint(i) + " medium flow rate",
					EntityType:        "number",
					UnitOfMeasurement: "m³/h",
					Max:               250,
					Min:               100,
					Step:              10,
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int * 10 }}",
					CommandTemplate:   "{{ (value / 10) | int }}",
				}
				m["D"+fmt.Sprint(i)+"_SPEED_HIGH"] = Attributes{
					Name:              "Fan " + fmt.Sprint(i) + " high flow rate",
					EntityType:        "number",
					UnitOfMeasurement: "m³/h",
					Max:               250,
					Min:               100,
					Step:              10,
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int * 10 }}",
					CommandTemplate:   "{{ (value / 10) | int }}",
				}
				m["D"+fmt.Sprint(i)+"_SPEED_BOOST"] = Attributes{
					Name:              "Fan " + fmt.Sprint(i) + " boost flow rate",
					EntityType:        "number",
					UnitOfMeasurement: "m³/h",
					Max:               250,
					Min:               100,
					Step:              10,
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int * 10 }}",
					CommandTemplate:   "{{ (value / 10) | int }}",
				}
				m["D"+fmt.Sprint(i)+"_SPEED_ECONOMY"] = Attributes{
					Name:              "Fan " + fmt.Sprint(i) + " economy flow rate",
					EntityType:        "number",
					UnitOfMeasurement: "m³/h",
					Max:               250,
					Min:               100,
					Step:              10,
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int * 10 }}",
					CommandTemplate:   "{{ (value / 10) | int }}",
				}
				m["D"+fmt.Sprint(i)+"_SPEED_COMFORT"] = Attributes{
					Name:              "Fan " + fmt.Sprint(i) + " comfort flow rate",
					EntityType:        "number",
					UnitOfMeasurement: "m³/h",
					Max:               250,
					Min:               100,
					Step:              10,
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int * 10 }}",
					CommandTemplate:   "{{ (value / 10) | int }}",
				}
			}
		}
	}
}

func (m ParamsMap) addEnergymeters(from map[string]string, static, read, write bool) {
	for i := 1; i <= 4; i++ {
		if static {
			m["EM"+fmt.Sprint(i)+"_INSTANT"] = Attributes{
				Name:              "Energy meter " + fmt.Sprint(i) + " power",
				EntityType:        "sensor",
				DeviceClass:       "power",
				UnitOfMeasurement: "kW",
				StateClass:        "measurement",
				ValueTemplate:     "{% if value | int >= 32768 %}{{ ((value | int) - 65536 ) / 100 }}{% else %}{{ value | int / 100 }}{% endif %}",
				CommandTemplate:   "",
			}
			m["EM"+fmt.Sprint(i)+"_ACCLO"] = Attributes{
				Name:              "Energy meter " + fmt.Sprint(i) + " total energy import",
				EntityType:        "sensor",
				DeviceClass:       "energy",
				UnitOfMeasurement: "kWh",
				StateClass:        "total_increasing",
				ValueTemplate:     "{{ value | int / 10 }}",
				CommandTemplate:   "",
			}
			// High word of the import accumulator. ACCLO wraps at 6553.5 kWh;
			// the true total is (ACCHI*65536+ACCLO)/10. The add-on publishes one
			// value per topic, so this is exposed as a raw diagnostic; combine it
			// with ACCLO in a Home Assistant template sensor. Gated on the meter
			// being present so sentinel (32769) phantom meters stay hidden.
			if from["EM"+fmt.Sprint(i)+"_ACCLO"] != "32769" {
				m["EM"+fmt.Sprint(i)+"_ACCHI"] = Attributes{
					Name:           "Energy meter " + fmt.Sprint(i) + " total energy import (high word)",
					EntityType:     "sensor",
					EntityCategory: "diagnostic",
					ValueTemplate:  "{{ value | int }}",
				}
			}
			if i == 4 {
				m["EM"+fmt.Sprint(i)+"_ACC2LO"] = Attributes{
					Name:              "Energy meter " + fmt.Sprint(i) + " total energy export",
					EntityType:        "sensor",
					DeviceClass:       "energy",
					UnitOfMeasurement: "kWh",
					StateClass:        "total_increasing",
					ValueTemplate:     "{{ value | int / 10 }}",
					CommandTemplate:   "",
				}
				if from["EM"+fmt.Sprint(i)+"_ACC2LO"] != "32769" {
					m["EM"+fmt.Sprint(i)+"_ACC2HI"] = Attributes{
						Name:           "Energy meter " + fmt.Sprint(i) + " total energy export (high word)",
						EntityType:     "sensor",
						EntityCategory: "diagnostic",
						ValueTemplate:  "{{ value | int }}",
					}
				}
			}
		}

	}
}

func (m ParamsMap) addCalendars(from map[string]string, static, read, write bool) {
	for i := 1; i <= 8; i++ {
		if from["MT"+fmt.Sprint(i)+"_XREF"] != "0" {
			if static {
				m["MT"+fmt.Sprint(i)+"_MODE"] = Attributes{
					Name:          "Calendar " + fmt.Sprint(i) + " mode",
					EntityType:    "sensor",
					DeviceClass:   "enum",
					ValueTemplate: "{% if value == \"1\" %}off{% elif value == \"2\" %}economy{% elif value == \"3\" %}comfort{% else %}{{ value }}{% endif %}",
				}
				m["MT"+fmt.Sprint(i)+"_XREF"] = Attributes{
					Name:           "Calendar " + fmt.Sprint(i) + " reference",
					EntityType:     "sensor",
					EntityCategory: "diagnostic",
					ValueTemplate:  "{{ value | int }}",
				}
			}
			if read {
				m["MT"+fmt.Sprint(i)+"_FORCING"] = Attributes{
					Name:          "Calendar " + fmt.Sprint(i) + " preset",
					EntityType:    "sensor",
					DeviceClass:   "enum",
					ValueTemplate: "{% if value == \"0\" %}automatic{% elif value == \"1\" %}forced off{% elif value == \"2\" %}forced economy{% elif value == \"3\" %}forced comfort{% else %}{{ value }}{% endif %}",
				}
			}
			if write {
				m["MT"+fmt.Sprint(i)+"_FORCING"] = Attributes{
					Name:            "Calendar " + fmt.Sprint(i) + " preset",
					Options:         []string{"automatic", "forced off", "forced economy", "forced comfort"},
					EntityType:      "select",
					ValueTemplate:   "{% if value == \"1\" %}forced off{% elif value == \"2\" %}forced economy{% elif value == \"3\" %}forced comfort{% else %}automatic{% endif %}",
					CommandTemplate: "{% if value == \"forced off\" %}1{% elif value == \"forced economy\" %}2{% elif value == \"forced comfort\" %}3{% else %}0{% endif %}",
				}

			}
		}
	}
}

// addAlarms exposes the station-wide alarm summary and the raw alarm bitmasks.
func (m ParamsMap) addAlarms(from map[string]string, static, read, write bool) {
	if static {
		if from["ANY_ALARM"] != "" {
			m["ANY_ALARM"] = Attributes{
				Name:          "Alarm active",
				EntityType:    "binary_sensor",
				DeviceClass:   "problem",
				ValueTemplate: "{% if value == \"0\" %}off{% else %}on{% endif %}",
			}
		}
		for _, s := range []string{"A", "B", "C"} {
			if from["ALARM_"+s] == "" {
				continue
			}
			m["ALARM_"+s] = Attributes{
				Name:           "Alarm bitmask " + s,
				EntityType:     "sensor",
				EntityCategory: "diagnostic",
				ValueTemplate:  "{{ value | int }}",
			}
		}
	}
}

// addSolar exposes the solar-thermal subsystem (sensors, pump, status). 255 on
// the pump output marks the solar function as not configured, so the whole
// family is gated on it and stays absent on stations without solar.
func (m ParamsMap) addSolar(from map[string]string, static, read, write bool) {
	if from["SOLAR_PUMP"] == "255" || from["SOLAR_PUMP"] == "" {
		return
	}
	if static {
		m["SOLAR_PUMP"] = Attributes{
			Name:          "Solar pump",
			EntityType:    "binary_sensor",
			ValueTemplate: "{% if value == \"1\" %}on{% else %}off{% endif %}",
		}
		m["SOLAR_STATUS"] = Attributes{
			Name:           "Solar status",
			EntityType:     "sensor",
			EntityCategory: "diagnostic",
			ValueTemplate:  "{{ value | int }}",
		}
		for i := 1; i <= 5; i++ {
			if from["SOLAR_S"+fmt.Sprint(i)] != "32769" {
				m["SOLAR_S"+fmt.Sprint(i)] = Attributes{
					Name:              "Solar temperature " + fmt.Sprint(i),
					EntityType:        "sensor",
					DeviceClass:       "temperature",
					UnitOfMeasurement: "°C",
					StateClass:        "measurement",
					ValueTemplate:     "{{ value | int / 10 }}",
				}
			}
		}
	}
}

// addDevice exposes the firmware/release identifiers as diagnostic sensors. The
// combined sw_version is also published in the device block (see SwVersion).
func (m ParamsMap) addDevice(from map[string]string, static, read, write bool) {
	if static {
		if from["FIRMWARE_RELEASE"] != "" {
			m["FIRMWARE_RELEASE"] = Attributes{
				Name:           "Firmware release",
				EntityType:     "sensor",
				EntityCategory: "diagnostic",
				ValueTemplate:  "{{ value | int }}",
			}
		}
		if from["DOT_RELEASE"] != "" {
			m["DOT_RELEASE"] = Attributes{
				Name:           "DOT release",
				EntityType:     "sensor",
				EntityCategory: "diagnostic",
				ValueTemplate:  "{{ value | int }}",
			}
		}
	}
}
