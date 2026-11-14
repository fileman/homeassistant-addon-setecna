package models

import "testing"

// Values taken from a live station with one OpenTherm boiler (generator 1).
func otStation() map[string]string {
	return map[string]string{
		"OT_G0_ENABLE": "0", "OT_G1_ENABLE": "17", "OT_G2_ENABLE": "0",
		"OT_G1_STATUS": "0", "OT_G1_RQ": "0", "OT_G1_POWER": "0",
		"OT_G1_TMAND": "317", "OT_G1_TRIT": "305", "OT_G1_RQTEMP": "0",
		"OT_G1_ERROR": "255", "OT_NCALD_ACTIVE": "0", "OT_R_SETPOINT": "0",
		"OT_A_SETPOINT": "0", "OT_SEC_TEMP": "347", "ACS_AUX_OUTPUT": "0",
	}
}

func TestOpenThermEntities(t *testing.T) {
	m := make(ParamsMap)
	m.AddEnabledParams(otStation(), true)

	for _, id := range []string{
		"ACS_AUX_OUTPUT", "OT_NCALD_ACTIVE", "OT_R_SETPOINT", "OT_A_SETPOINT",
		"OT_SEC_TEMP", "OT_G1_STATUS", "OT_G1_STATUS_FLAME", "OT_G1_STATUS_FAULT",
		"OT_G1_RQ", "OT_G1_POWER", "OT_G1_TMAND", "OT_G1_TRIT", "OT_G1_RQTEMP",
		"OT_G1_ERROR",
	} {
		if _, ok := m[id]; !ok {
			t.Errorf("missing entity %s", id)
		}
	}
	for _, id := range []string{"OT_G0_STATUS", "OT_G2_STATUS"} {
		if _, ok := m[id]; ok {
			t.Errorf("disabled generator exposed: %s", id)
		}
	}

	// The flame binary sensor reads the status word topic, not its own.
	var bs BinarySensor
	bs.Init("SYS", "OT_G1_STATUS_FLAME", m["OT_G1_STATUS_FLAME"])
	if want := "homeassistant/sensor/SYS_OT_G1_STATUS"; bs.StateTopic != want {
		t.Errorf("flame state topic = %q, want %q", bs.StateTopic, want)
	}
	if want := "SYS_OT_G1_STATUS_FLAME"; bs.UniqueID != want {
		t.Errorf("flame unique id = %q, want %q", bs.UniqueID, want)
	}
	// Ordinary entities keep their own topic.
	var s Sensor
	s.Init("SYS", "OT_G1_TMAND", m["OT_G1_TMAND"])
	if want := "homeassistant/sensor/SYS_OT_G1_TMAND"; s.StateTopic != want {
		t.Errorf("flow state topic = %q, want %q", s.StateTopic, want)
	}
}

func TestOpenThermAbsentWithoutGenerators(t *testing.T) {
	from := otStation()
	from["OT_G1_ENABLE"] = "0"
	m := make(ParamsMap)
	m.AddEnabledParams(from, true)
	for id := range m {
		if len(id) > 3 && id[:3] == "OT_" {
			t.Errorf("OpenTherm entity %s exposed with no generator enabled", id)
		}
	}
}
