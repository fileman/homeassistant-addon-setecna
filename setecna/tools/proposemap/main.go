// Command proposemap reads a Setecna getres dump (or the add-on's
// /share/setecna_unmapped_ids.json export) and proposes Home Assistant entity
// mappings for parameters the add-on does not yet map.
//
// It is deliberately SELF-CONTAINED: it does NOT import the project's models
// package, so it builds and runs even when the rest of the repository does not.
//
// Usage (from the setecna/ module root):
//
//	go run ./tools/proposemap <dumpfile>
//
// <dumpfile> may be EITHER
//
//	(a) the unmapped list  : a top-level JSON array of {"Id":"...","V":"..."}
//	    (as written by cmd/main.go to /share/setecna_unmapped_ids.json), OR
//	(b) the raw getres dump: an object {"Data":[{"Id":"...","V":...}], ...}
//	    where V may be a JSON number or a JSON string.
//
// The shape is auto-detected and both V encodings are handled.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// Input parsing
// ---------------------------------------------------------------------------

// param is one parameter read from the dump. V is normalised to a string so the
// two input encodings (number / string) are handled uniformly downstream.
type param struct {
	ID string
	V  string
}

// rawParam mirrors the on-the-wire shape. V is json.RawMessage so we can accept
// either a JSON number or a JSON string for the value.
type rawParam struct {
	ID string          `json:"Id"`
	V  json.RawMessage `json:"V"`
}

// getresDump is shape (b): the raw getres object with a Data array.
type getresDump struct {
	Data []rawParam `json:"Data"`
}

// normaliseV turns a json.RawMessage (string OR number OR null) into a plain
// string, mirroring how cmd/main.go does string(num.V).
func normaliseV(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return ""
	}
	// If it is a JSON string, unquote it. Otherwise it is a number/bool: use the
	// literal text as-is (this is what the add-on sees for numeric values).
	if len(s) > 0 && s[0] == '"' {
		var str string
		if err := json.Unmarshal(raw, &str); err == nil {
			return str
		}
	}
	return s
}

// parseInput auto-detects the two accepted shapes and returns a flat []param.
func parseInput(data []byte) ([]param, string, error) {
	trimmed := strings.TrimSpace(string(data))
	if len(trimmed) == 0 {
		return nil, "", fmt.Errorf("input file is empty")
	}

	switch trimmed[0] {
	case '[': // shape (a): top-level array of {Id,V}
		var arr []rawParam
		if err := json.Unmarshal(data, &arr); err != nil {
			return nil, "", fmt.Errorf("parsing unmapped-list array: %w", err)
		}
		out := make([]param, 0, len(arr))
		for _, p := range arr {
			out = append(out, param{ID: p.ID, V: normaliseV(p.V)})
		}
		return out, "unmapped-list array ([{Id,V}])", nil

	case '{': // shape (b): getres object with a Data array
		var dump getresDump
		if err := json.Unmarshal(data, &dump); err != nil {
			return nil, "", fmt.Errorf("parsing getres dump object: %w", err)
		}
		out := make([]param, 0, len(dump.Data))
		for _, p := range dump.Data {
			out = append(out, param{ID: p.ID, V: normaliseV(p.V)})
		}
		return out, "raw getres dump ({Data:[{Id,V}]})", nil

	default:
		return nil, "", fmt.Errorf("unrecognised JSON top-level token %q (expected '[' or '{')", trimmed[0])
	}
}

// ---------------------------------------------------------------------------
// Families
// ---------------------------------------------------------------------------

// hasIndexedPrefix reports whether id begins with the single-letter family
// prefix p immediately followed by a digit (e.g. "Z3_TEMP" for "Z"). This keeps
// non-indexed IDs that merely start with the same letter — SW_VERSION, SERIAL,
// DEWPOINT, CO2_... — out of the Z/C/S/D families and in the "other" bucket.
func hasIndexedPrefix(id, p string) bool {
	return strings.HasPrefix(id, p) && len(id) > len(p) && id[len(p)] >= '0' && id[len(p)] <= '9'
}

// familyOf classifies an ID by its prefix into one of the known families, or
// "other" if nothing matches. Order matters: longer / more specific prefixes
// (FALDIN, FAIN, FDIN) must be tested before the single-letter ones (F is not a
// family, but D/S/C would otherwise swallow longer prefixes). The single-letter
// families additionally require a digit after the letter (see hasIndexedPrefix).
func familyOf(id string) string {
	switch {
	case strings.HasPrefix(id, "EM"):
		return "EM"
	case strings.HasPrefix(id, "FALDIN"):
		return "FALDIN"
	case strings.HasPrefix(id, "FAIN"):
		return "FAIN"
	case strings.HasPrefix(id, "FDIN"):
		return "FDIN"
	case strings.HasPrefix(id, "GLOBAL"):
		return "GLOBAL"
	case strings.HasPrefix(id, "ACS"):
		return "ACS"
	case strings.HasPrefix(id, "MT"):
		return "MT"
	case hasIndexedPrefix(id, "Z"):
		return "Z"
	case hasIndexedPrefix(id, "C"):
		return "C"
	case hasIndexedPrefix(id, "S"):
		return "S"
	case hasIndexedPrefix(id, "D"):
		return "D"
	default:
		return "other"
	}
}

// familyOrder is the print order; "other" always comes last.
var familyOrder = []string{
	"EM", "FAIN", "FDIN", "FALDIN", "Z", "C", "S", "D", "MT", "GLOBAL", "ACS", "other",
}

var familyTitle = map[string]string{
	"EM":     "EM  — Energy meters",
	"FAIN":   "FAIN — Analog inputs",
	"FDIN":   "FDIN — Digital inputs",
	"FALDIN": "FALDIN — Digital alarms",
	"Z":      "Z  — Zones",
	"C":      "C  — Circuits",
	"S":      "S  — Sources",
	"D":      "D  — Dehumidifiers",
	"MT":     "MT — Calendars / timetables",
	"GLOBAL": "GLOBAL — System-wide params",
	"ACS":    "ACS — Domestic hot water",
	"other":  "other — Unclassified bucket",
}

// ---------------------------------------------------------------------------
// Already-mapped ID detection (warn if a supposedly-unmapped ID is mapped)
// ---------------------------------------------------------------------------

// mappedExact lists IDs the add-on maps verbatim (no index). Derived from
// models/attributes.go (globals, DHW, last-update).
var mappedExact = map[string]bool{
	"LAST_UPDATE":             true,
	"GLOBAL_ENABLE":           true,
	"GLOBAL_T_EXT":            true,
	"GLOBAL_SEASON":           true,
	"GLOBAL_DEICING":          true,
	"GLOBAL_EXPECTED_DEWP":    true,
	"GLOBAL_ZONE_T_HYST":      true,
	"GLOBAL_ZONE_RH_HYST":     true,
	"GLOBAL_ZONE_DEICE_TRESH": true,
	"ACS_MAIN_OUTPUT":         true,
	"GLOBAL_ACS_ENABLE":       true,
	"GLOBAL_T_ACS":            true,
	"GLOBAL_SET_ACS":          true,
	"ACS_SET_ECONOMY":         true,
	"ACS_SET_COMFORT":         true,
	"ACS_SET_DELTA":           true,
}

// mappedBases is the set of index-stripped IDs the add-on actually maps inside
// its indexed loops, e.g. Z{n}_OUTPUT -> "Z_OUTPUT", EM{n}_ACCLO -> "EM_ACCLO".
// Matching is EXACT on the stripped base (not a suffix test), which is what the
// add-on really does — it maps specific IDs, not "anything ending in _OUTPUT".
// So a proposed Z{n}_RH_OUTPUT is NOT mistaken for the mapped Z{n}_OUTPUT, and a
// proposed C{n}_OUTPUT is not mistaken for anything (circuits map only
// _TEMP/_SET). Derived directly from models/attributes.go. Gate-only reads the
// add-on never surfaces (S{n}_DESCR, MT{n}_XREF, Z{n}_SENSOR_CHN) are
// deliberately absent — they are legitimate unmapped params.
var mappedBases = map[string]bool{
	"FAIN_TEMP":       true, // addAnalogInput
	"FDIN_STATUS":     true, // addDigitalInput
	"FALDIN_STATUS":   true, // addDigitalAlarm
	"Z_OUTPUT":        true, // addZones
	"Z_TEMP":          true,
	"Z_ZONE_MODE":     true,
	"Z_ZONE_SET":      true,
	"Z_FORCING":       true,
	"Z_SET_CW":        true,
	"Z_SET_EW":        true,
	"Z_SET_CS":        true,
	"Z_SET_ES":        true,
	"Z_RH":            true,
	"Z_SET_RH":        true,
	"C_TEMP":          true, // addCircuits
	"C_SET":           true,
	"S_ENABLED":       true, // addSources (NOT S_DESCR — gate only)
	"S_OUTPUT":        true,
	"S_AUXOUTPUT":     true,
	"D_OUTPUT_RENEW":  true, // addDehumidifier
	"D_OUTPUT_DEUM":   true,
	"D_SPEED_LOW":     true,
	"D_SPEED_MED":     true,
	"D_SPEED_HIGH":    true,
	"D_SPEED_BOOST":   true,
	"D_SPEED_ECONOMY": true,
	"D_SPEED_COMFORT": true,
	"EM_INSTANT":      true, // addEnergymeters
	"EM_ACCLO":        true,
	// EM_ACC2LO is mapped for EM4 ONLY (attributes.go i==4 guard) — handled in
	// isAlreadyMapped with an index check, not listed here.
	"MT_MODE":    true, // addCalendars
	"MT_FORCING": true,
}

// isAlreadyMapped reports whether the add-on already emits an entity for id.
// The check is EXACT on the index-stripped base, so it is inherently
// family-aware and never confuses a longer ID (Z{n}_RH_OUTPUT) with a mapped
// shorter one (Z{n}_OUTPUT). Gate-only reads (S{n}_DESCR, MT{n}_XREF,
// Z{n}_SENSOR_CHN) are read but not surfaced, so they are NOT treated as mapped.
func isAlreadyMapped(id string) bool {
	if mappedExact[id] {
		return true
	}
	base := stripIndex(id)
	// EM{n}_ACC2LO (export low word) is mapped for EM4 only (attributes.go:741).
	if base == "EM_ACC2LO" {
		return indexOf(id) == "4"
	}
	return mappedBases[base]
}

// stripIndex removes the leading numeric index that follows a family prefix,
// e.g. "FAIN12_TEMP" -> "FAIN_TEMP", "EM4_ACC2LO" -> "EM_ACC2LO", so suffix
// matching is index-agnostic. IDs with no index are returned unchanged.
func stripIndex(id string) string {
	for i := 0; i < len(id); i++ {
		if id[i] >= '0' && id[i] <= '9' {
			j := i
			for j < len(id) && id[j] >= '0' && id[j] <= '9' {
				j++
			}
			return id[:i] + id[j:]
		}
	}
	return id
}

// indexOf extracts the first run of digits in an ID (the family index), or ""
// if there is none. "FAIN3_DESCR" -> "3".
func indexOf(id string) string {
	start := -1
	for i := 0; i < len(id); i++ {
		if id[i] >= '0' && id[i] <= '9' {
			start = i
			break
		}
	}
	if start == -1 {
		return ""
	}
	end := start
	for end < len(id) && id[end] >= '0' && id[end] <= '9' {
		end++
	}
	return id[start:end]
}

// ---------------------------------------------------------------------------
// Proposal model + per-family heuristics
// ---------------------------------------------------------------------------

type proposal struct {
	id              string
	sampleValue     string
	name            string
	entityType      string
	deviceClass     string
	unit            string
	valueTemplate   string
	scaling         string
	confidence      string // high | medium | low | none
	confirmWithDump bool
	note            string
}

// confidenceRank is used to sort proposals (high first) and to decide which go
// into the ready-to-paste snippet.
func confidenceRank(c string) int {
	switch c {
	case "high":
		return 0
	case "medium":
		return 1
	case "low":
		return 2
	default:
		return 3
	}
}

// classify applies the per-family heuristics to a single parameter and returns
// a proposal. An empty entityType means "unclassified / needs manual review".
func classify(p param) proposal {
	id := p.ID
	fam := familyOf(id)
	idx := indexOf(id)
	base := stripIndex(id)

	pr := proposal{id: id, sampleValue: p.V}

	switch fam {
	case "EM":
		// EM{n}_ACCHI / EM4_ACC2HI — high-word companions of the LOW-word
		// accumulators the add-on already maps. See the energy backlog item.
		switch {
		case strings.HasSuffix(base, "_ACCHI"):
			pr.name = fmt.Sprintf("Energy meter %s total energy import (high word)", orQ(idx))
			pr.entityType = "sensor"
			pr.deviceClass = "energy"
			pr.unit = "kWh"
			pr.valueTemplate = "{{ value | int * 6553.6 }}"
			pr.scaling = "high 16-bit word of the import accumulator; each high count = 65536 raw low units = 6553.6 kWh (the EM" + orQ(idx) + "_ACCLO entity is raw_low/10; in the combine formula use the RAW low word, not the divided entity value). Pairs with EM" + orQ(idx) + "_ACCLO."
			pr.confidence = "medium"
			pr.confirmWithDump = true
			pr.note = "value_template cannot reference ACCLO (other topic); expose the high word standalone. True total (ACCHI*65536+ACCLO)/10 must be assembled HA-side or pre-combined in the add-on. StateClass total_increasing vs measurement is an open question."
		case strings.HasSuffix(base, "_ACC2HI"):
			pr.name = fmt.Sprintf("Energy meter %s total energy export (high word)", orQ(idx))
			pr.entityType = "sensor"
			pr.deviceClass = "energy"
			pr.unit = "kWh"
			pr.valueTemplate = "{{ value | int * 6553.6 }}"
			pr.scaling = "high 16-bit word of the EM" + orQ(idx) + " export accumulator; pairs with the existing EM4_ACC2LO low word (/10). Emit only when index==4, matching the ACC2LO guard."
			pr.confidence = "medium"
			pr.confirmWithDump = true
			pr.note = "Same value_template-scope caveat as ACCHI: standalone high word, not the combined total. Confirm whether export accumulators exist for EM1..EM3 or only EM4."
		default:
			pr.unclassified()
		}

	case "FAIN":
		switch {
		case strings.HasSuffix(base, "_DESCR"):
			pr.name = fmt.Sprintf("Analog input %s type", orQ(idx))
			pr.entityType = "sensor"
			pr.deviceClass = "enum"
			pr.valueTemplate = `{% if value == "0" %}temperature{% elif value == "1" %}humidity{% elif value == "2" %}pressure{% elif value == "3" %}co2{% else %}{{ value }}{% endif %}`
			pr.scaling = "raw enum code, read-only. Exact codes are a guess — the live dump must confirm the numbering."
			pr.confidence = "low"
			pr.confirmWithDump = true
			pr.note = "Mirrors the S{n}_DESCR descriptor-gating precedent (attributes.go:566). Both the ID spelling (FAIN{n}_DESCR vs _TYPE/_KIND) and the code-to-label table are unconfirmed."
		case strings.HasSuffix(base, "_TEMP"):
			// Already mapped as temperature; the backlog asks to make it dynamic.
			pr.name = fmt.Sprintf("Analog input %s", orQ(idx))
			pr.entityType = "sensor"
			pr.deviceClass = "(dynamic: temperature|humidity|pressure|carbon_dioxide from FAIN{n}_DESCR)"
			pr.unit = "(dynamic: °C|%|hPa|ppm)"
			pr.valueTemplate = "{{ value | int / 10 }}"
			pr.scaling = "/10 read-only (unchanged). CHANGE: pick DeviceClass/Unit from FAIN{n}_DESCR instead of hardcoding temperature/°C. Gate stays from[FAIN{n}_TEMP] != 32769."
			pr.confidence = "medium"
			pr.confirmWithDump = true
			pr.note = "BACKLOG ITEM: all FAIN* hardcoded as temperature/°C is wrong for humidity/pressure/etc. Default to temperature/°C when descriptor absent so confirmed probes are unchanged. /10 uniformity across probe types is unverified."
		default:
			pr.unclassified()
		}

	case "Z":
		switch {
		case strings.HasSuffix(base, "_SENSOR_CHN"):
			pr.name = fmt.Sprintf("Zone %s sensor channel", orQ(idx))
			pr.entityType = "sensor"
			pr.valueTemplate = "{{ value | int }}"
			pr.scaling = "raw integer channel index, no /10."
			pr.confidence = "high"
			pr.confirmWithDump = false
			pr.note = "PROVABLY PRESENT: it is the addZones gate (attributes.go:362) and the CreateClimates/RemoveClimates gate (homeassistant.go:22,25,58), read but never surfaced. Set EntityCategory:\"diagnostic\" INLINE — markDiagnostics() won't auto-tag it. Place inside the existing Z{n}_SENSOR_CHN!=0 / if static block."
		case strings.HasSuffix(base, "_RH_OUTPUT") || strings.HasSuffix(base, "_DEUM_OUTPUT"):
			pr.name = fmt.Sprintf("Zone %s dehumidify state", orQ(idx))
			pr.entityType = "binary_sensor"
			pr.valueTemplate = `{% if value == "1" %}on{% else %}off{% endif %}`
			pr.scaling = `value=="1" -> on else off`
			pr.confidence = "medium"
			pr.confirmWithDump = true
			pr.note = "Mirrors Z{n}_OUTPUT (attributes.go:364) and D{n}_OUTPUT_DEUM (line 597). Gate under from[Z{n}_RH]!=32769. Exact spelling (Z{n}_RH_OUTPUT / _DEUM_OUTPUT / _OUTPUT_DEUM) is a guess."
		case strings.HasSuffix(base, "_DEWP") || strings.HasSuffix(base, "_DEWPOINT") || strings.HasSuffix(base, "_EXPECTED_DEWP"):
			pr.name = fmt.Sprintf("Zone %s dewpoint", orQ(idx))
			pr.entityType = "sensor"
			pr.deviceClass = "temperature"
			pr.unit = "°C"
			pr.valueTemplate = "{{ value | int / 10 }}"
			pr.scaling = "/10, mirrors GLOBAL_EXPECTED_DEWP (attributes.go:128-135)."
			pr.confidence = "low"
			pr.confirmWithDump = true
			pr.note = "Plausible per-zone condensation control value; gate under from[Z{n}_RH]!=32769. ID name is a pure guess — confirm against the dump."
		default:
			pr.unclassified()
		}

	case "S":
		switch {
		case strings.HasSuffix(base, "_DESCR"):
			pr.name = fmt.Sprintf("Source %s type", orQ(idx))
			pr.entityType = "sensor"
			pr.deviceClass = "enum"
			pr.valueTemplate = "{{ value }}"
			pr.scaling = "raw enum/descriptor code, read-only. Currently read as a GATE (from[S{n}_DESCR]!=0, attributes.go:566) but not surfaced as an entity."
			pr.confidence = "medium"
			pr.confirmWithDump = true
			pr.note = "The source family already keys on this descriptor; exposing it as an enum sensor (with a confirmed code-to-label table) is the natural follow-up. Codes unknown until the dump lists them."
		case strings.HasSuffix(base, "_FAULT") || strings.HasSuffix(base, "_ALARM") || strings.HasSuffix(base, "_ERROR"):
			pr.name = fmt.Sprintf("Source %s fault", orQ(idx))
			pr.entityType = "binary_sensor"
			pr.deviceClass = "problem"
			pr.valueTemplate = `{% if value == "1" %}on{% else %}off{% endif %}`
			pr.scaling = `value=="1" -> on else off`
			pr.confidence = "medium"
			pr.confirmWithDump = true
			pr.note = "Source diagnostic; mirrors the on/off binary pattern of S{n}_OUTPUT. device_class=problem is the HA convention for fault flags. Likely a diagnostic entity. Gate under from[S{n}_DESCR]!=0."
		default:
			pr.unclassified()
		}

	case "C":
		switch {
		case strings.HasSuffix(base, "_PUMP") || strings.HasSuffix(base, "_OUTPUT"):
			pr.name = fmt.Sprintf("Circuit %s pump state", orQ(idx))
			pr.entityType = "binary_sensor"
			pr.valueTemplate = `{% if value == "1" %}on{% else %}off{% endif %}`
			pr.scaling = `value=="1" -> on else off`
			pr.confidence = "medium"
			pr.confirmWithDump = true
			pr.note = "Circuits already map C{n}_TEMP / C{n}_SET (gated on C{n}_TEMP!=32769, attributes.go:539). A per-circuit pump/output binary is plausible; gate under the same 32769 guard."
		default:
			pr.unclassified()
		}

	case "MT":
		switch {
		case strings.HasSuffix(base, "_XREF"):
			pr.name = fmt.Sprintf("Calendar %s reference", orQ(idx))
			pr.entityType = "sensor"
			pr.valueTemplate = "{{ value | int }}"
			pr.scaling = "raw integer reference index, no /10."
			pr.confidence = "medium"
			pr.confirmWithDump = true
			pr.note = "MT{n}_XREF is the calendar GATE (from[MT{n}_XREF]!=0, attributes.go:759), read but not surfaced. Expose as a diagnostic sensor like the proposed Z{n}_SENSOR_CHN; set EntityCategory:\"diagnostic\" inline."
		default:
			pr.unclassified()
		}

	case "FDIN", "FALDIN":
		// These families ARE mapped (FDIN{n}_STATUS / FALDIN{n}_STATUS). Anything
		// here that is not _STATUS is a new readout; flag for manual review.
		pr.unclassified()

	case "D":
		// Dehumidifier outputs/speeds are mapped; anything else is unknown.
		pr.unclassified()

	case "GLOBAL", "ACS":
		// Most GLOBAL_/ACS_ params are mapped exactly; new ones need review.
		pr.unclassified()

	default: // "other"
		pr.unclassified()
	}

	return pr
}

// unclassified marks a proposal as needing manual review.
func (pr *proposal) unclassified() {
	pr.entityType = ""
	pr.confidence = "none"
	pr.name = "UNCLASSIFIED — needs manual review"
}

// orQ returns the index, or "{n}" when there is no numeric index, so names read
// sensibly for both indexed and non-indexed IDs.
func orQ(idx string) string {
	if idx == "" {
		return "{n}"
	}
	return idx
}

// ---------------------------------------------------------------------------
// Output
// ---------------------------------------------------------------------------

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: go run ./tools/proposemap <dumpfile>\n")
		os.Exit(2)
	}
	path := os.Args[1]

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: reading %s: %v\n", path, err)
		os.Exit(1)
	}

	params, shape, err := parseInput(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Classify everything, bucketed by family.
	byFamily := map[string][]proposal{}
	var warnings []proposal // IDs that are actually already-mapped
	for _, p := range params {
		if isAlreadyMapped(p.ID) {
			warnings = append(warnings, classify(p))
			// still classify+group below so it appears in its family section too,
			// but mark it; we handle the warning banner separately.
		}
		fam := familyOf(p.ID)
		byFamily[fam] = append(byFamily[fam], classify(p))
	}

	fmt.Println("===========================================================================")
	fmt.Println(" Setecna unmapped-parameter mapping proposals")
	fmt.Println("===========================================================================")
	fmt.Printf(" Input file   : %s\n", path)
	fmt.Printf(" Detected shape: %s\n", shape)
	fmt.Printf(" Parameters   : %d\n", len(params))
	fmt.Println("===========================================================================")
	fmt.Println()

	// Stale-dump / already-mapped warnings up front.
	if len(warnings) > 0 {
		fmt.Println("!! WARNING: the following input IDs appear to be ALREADY MAPPED by the add-on.")
		fmt.Println("!! If they show up as unmapped, the dump is stale or the gating changed.")
		sort.Slice(warnings, func(i, j int) bool { return warnings[i].id < warnings[j].id })
		for _, w := range warnings {
			fmt.Printf("   - %-22s (sample V=%q)\n", w.id, w.sampleValue)
		}
		fmt.Println()
	}

	// Per-family sections in canonical order.
	var highConf []proposal
	var unclassified []proposal
	famCounts := map[string]int{}

	for _, fam := range familyOrder {
		props := byFamily[fam]
		if len(props) == 0 {
			continue
		}
		famCounts[fam] = len(props)

		sort.Slice(props, func(i, j int) bool {
			if confidenceRank(props[i].confidence) != confidenceRank(props[j].confidence) {
				return confidenceRank(props[i].confidence) < confidenceRank(props[j].confidence)
			}
			return props[i].id < props[j].id
		})

		title := familyTitle[fam]
		if title == "" {
			title = fam
		}
		fmt.Println("---------------------------------------------------------------------------")
		fmt.Printf(" %s  (%d)\n", title, len(props))
		fmt.Println("---------------------------------------------------------------------------")

		for _, pr := range props {
			if pr.entityType == "" {
				fmt.Printf("  [UNCLASSIFIED] %s\n", pr.id)
				fmt.Printf("        sample V : %q\n", pr.sampleValue)
				fmt.Printf("        note     : UNCLASSIFIED — needs manual review\n\n")
				unclassified = append(unclassified, pr)
				continue
			}
			mappedTag := ""
			if isAlreadyMapped(pr.id) {
				mappedTag = "  [!! already mapped — see warning above]"
			}
			fmt.Printf("  %s  (confidence=%s)%s\n", pr.id, pr.confidence, mappedTag)
			fmt.Printf("        name        : %s\n", pr.name)
			fmt.Printf("        sample V    : %q\n", pr.sampleValue)
			fmt.Printf("        entity_type : %s\n", pr.entityType)
			if pr.deviceClass != "" {
				fmt.Printf("        device_class: %s\n", pr.deviceClass)
			}
			if pr.unit != "" {
				fmt.Printf("        unit        : %s\n", pr.unit)
			}
			if pr.valueTemplate != "" {
				fmt.Printf("        value_tmpl  : %s\n", pr.valueTemplate)
			}
			fmt.Printf("        scaling     : %s\n", pr.scaling)
			fmt.Printf("        confirm dump: %v\n", pr.confirmWithDump)
			if pr.note != "" {
				fmt.Printf("        note        : %s\n", pr.note)
			}
			fmt.Println()

			if pr.confidence == "high" {
				highConf = append(highConf, pr)
			}
		}
	}

	// Summary counts.
	fmt.Println("===========================================================================")
	fmt.Println(" SUMMARY")
	fmt.Println("===========================================================================")
	total := 0
	for _, fam := range familyOrder {
		if c := famCounts[fam]; c > 0 {
			title := familyTitle[fam]
			if title == "" {
				title = fam
			}
			fmt.Printf("  %-32s %3d\n", title, c)
			total += c
		}
	}
	fmt.Printf("  %-32s %3d\n", "TOTAL", total)
	fmt.Printf("  %-32s %3d\n", "high-confidence proposals", len(highConf))
	fmt.Printf("  %-32s %3d\n", "UNCLASSIFIED (manual review)", len(unclassified))
	fmt.Printf("  %-32s %3d\n", "already-mapped warnings", len(warnings))
	fmt.Println()

	// Ready-to-paste Go snippet for the high-confidence proposals.
	emitSnippet(highConf)
}

// emitSnippet prints a ready-to-paste Go Attributes{} block for the
// high-confidence proposals, matching the literal style used in attributes.go.
func emitSnippet(props []proposal) {
	fmt.Println("===========================================================================")
	fmt.Println(" READY-TO-PASTE Attributes{} SNIPPET (high-confidence proposals only)")
	fmt.Println("===========================================================================")
	if len(props) == 0 {
		fmt.Println("  (no high-confidence proposals — nothing to paste)")
		return
	}
	sort.Slice(props, func(i, j int) bool { return props[i].id < props[j].id })

	fmt.Println("// Add inside the relevant add* function in models/attributes.go.")
	fmt.Println("// Indexed IDs use fmt.Sprint(i); adjust the gate to match the family loop.")
	fmt.Println()
	for _, pr := range props {
		idLiteral := goIDLiteral(pr.id)
		fmt.Printf("m[%s] = Attributes{\n", idLiteral)
		fmt.Printf("\tName:          %s,\n", goNameLiteral(pr.id, pr.name))
		fmt.Printf("\tEntityType:    %s,\n", strconv.Quote(pr.entityType))
		if pr.deviceClass != "" && !strings.HasPrefix(pr.deviceClass, "(") {
			fmt.Printf("\tDeviceClass:   %s,\n", strconv.Quote(pr.deviceClass))
		}
		if pr.unit != "" && !strings.HasPrefix(pr.unit, "(") {
			fmt.Printf("\tUnitOfMeasurement: %s,\n", strconv.Quote(pr.unit))
		}
		if pr.valueTemplate != "" {
			fmt.Printf("\tValueTemplate: %s,\n", strconv.Quote(pr.valueTemplate))
		}
		// High-confidence diagnostic IDs (channel/reference indexes) need the
		// inline diagnostic category, since markDiagnostics() won't tag them.
		if needsInlineDiagnostic(pr.id) {
			fmt.Printf("\tEntityCategory: \"diagnostic\", // markDiagnostics() will NOT auto-tag this id\n")
		}
		fmt.Printf("}\n")
		fmt.Printf("// scaling: %s\n", pr.scaling)
		fmt.Println()
	}
}

// goIDLiteral renders the map key as a Go expression. For indexed IDs it builds
// the "PREFIX"+fmt.Sprint(i)+"_SUFFIX" form used throughout attributes.go.
func goIDLiteral(id string) string {
	idx := indexOf(id)
	if idx == "" {
		return strconv.Quote(id)
	}
	pos := strings.Index(id, idx)
	prefix := id[:pos]
	suffix := id[pos+len(idx):]
	return fmt.Sprintf("%s+fmt.Sprint(i)+%s", strconv.Quote(prefix), strconv.Quote(suffix))
}

// goNameLiteral renders the entity Name for the pasteable snippet. For an
// indexed ID whose name embeds that same index (e.g. "Zone 1 sensor channel"
// for Z1_SENSOR_CHN), it parameterizes the index as fmt.Sprint(i) so the
// snippet reads as a loop body consistent with the map key. Otherwise it is a
// plain quoted string.
func goNameLiteral(id, name string) string {
	idx := indexOf(id)
	if idx == "" {
		return strconv.Quote(name)
	}
	// Only parameterize when the name contains the index as a standalone token
	// (surrounded by spaces or at the ends) to avoid mangling unrelated digits.
	tok := " " + idx + " "
	if strings.Contains(" "+name+" ", tok) {
		i := strings.Index(name, idx)
		// guard against re-quoting if the index is embedded in a larger number
		return fmt.Sprintf("%s+fmt.Sprint(i)+%s",
			strconv.Quote(name[:i]), strconv.Quote(name[i+len(idx):]))
	}
	return strconv.Quote(name)
}

// needsInlineDiagnostic reports whether the id is a gate-style index that
// markDiagnostics() would not auto-tag (sensor channel / calendar xref).
func needsInlineDiagnostic(id string) bool {
	base := stripIndex(id)
	return strings.HasSuffix(base, "_SENSOR_CHN") || strings.HasSuffix(base, "_XREF")
}
