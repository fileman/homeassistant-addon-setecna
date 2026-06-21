// Command verifymap loads a raw getres dump, builds the Home Assistant entity
// map exactly like the add-on does, and reports the newly-added entities and
// the decoded (free-text) names so the enrichment can be verified end-to-end.
//
//	go run ./tools/verifymap tools/proposemap/testdata/live_getres.json
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Ingordigia/homeassistant-addon-setecna/models"
)

type rawParam struct {
	ID string          `json:"Id"`
	V  json.RawMessage `json:"V"`
}
type dump struct {
	Data []rawParam `json:"Data"`
}

func norm(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return ""
	}
	if s[0] == '"' {
		var str string
		if json.Unmarshal(raw, &str) == nil {
			return str
		}
	}
	return s
}

func main() {
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var d dump
	if err := json.Unmarshal(data, &d); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	from := map[string]string{}
	for _, p := range d.Data {
		from[p.ID] = norm(p.V)
	}

	// Mirror the add-on: readonly deployment builds the enabled (read) entity set.
	m := make(models.ParamsMap)
	m.AddEnabledParams(from, true)

	fmt.Printf("sw_version device block : %q\n", "FW "+from["FIRMWARE_RELEASE"]+" / DOT "+from["DOT_RELEASE"])
	fmt.Printf("total entities built     : %d\n\n", len(m))

	// Decoded names (the _FREEDESC decoder in action).
	fmt.Println("== Decoded names (zones / circuits / sources / analog) ==")
	for _, pref := range []string{"Z", "C", "S", "FAIN"} {
		var ids []string
		for id := range m {
			if strings.HasPrefix(id, pref) {
				ids = append(ids, id)
			}
		}
		sort.Strings(ids)
		for _, id := range ids {
			n := m[id].Name
			// Only show the "head" entities that carry the decoded base label.
			if strings.Contains(id, "_OUTPUT") || strings.Contains(id, "_TEMP") || strings.HasSuffix(id, "_ENABLED") {
				fmt.Printf("  %-16s -> %q\n", id, n)
			}
		}
	}

	// New families/entities introduced by this change.
	fmt.Println("\n== New entities present ==")
	checks := []string{
		"Z5_SENSOR_CHN", "Z5_DEUM", "Z5_TEMP_OFFSET", "Z5_LINKED",
		"C1_OUTPUT", "C1_MODE", "S1_TEMP", "S1_PRIORITY", "S1_OUTPUT_010",
		"MT1_XREF", "ANY_ALARM", "ALARM_A", "FIRMWARE_RELEASE", "DOT_RELEASE",
		"Z5_DEWPOINT", "SOLAR_PUMP", "EM1_ACCHI",
	}
	for _, id := range checks {
		if a, ok := m[id]; ok {
			fmt.Printf("  [present] %-18s name=%q type=%s cat=%s\n", id, a.Name, a.EntityType, a.EntityCategory)
		} else {
			fmt.Printf("  [absent ] %-18s (gated out — expected if sentinel/not installed)\n", id)
		}
	}
}
