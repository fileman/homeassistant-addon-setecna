<!-- https://developers.home-assistant.io/docs/add-ons/presentation#keeping-a-changelog -->

## 1.3.0

Data-enrichment release — maps many parameters the station already reported but
the add-on ignored. Derived from a live `getres` capture; new families are gated
on presence, so stations without the relevant hardware see no extra entities.

- **Entities are named from your station labels.** A zone/circuit/source/analog
  input whose description points at a free-text label (`_FREEDESC`) now uses that
  label as its name — the name you set in the Setecna web-interface — instead of
  the generic *"Zone N"*. Climate entities are renamed too. Entity IDs are unchanged
  (only the friendly name), so existing automations/dashboards keep working.
- **Device card** now shows `sw_version` (from `FIRMWARE_RELEASE` / `DOT_RELEASE`),
  plus diagnostic *Firmware release* / *DOT release* sensors.
- **Zones:** added sensor channel, dehumidify demand, temperature offset, linked
  zone (diagnostic) and dewpoint (when a dewpoint probe is present).
- **Circuits:** now appear when they have a descriptor even without a temperature
  probe; added the circuit pump state and mode.
- **Sources:** added flow / auxiliary temperature, priority and 0-10V modulation.
- **Calendars:** added the program reference (diagnostic).
- **Alarms:** added an *Alarm active* binary sensor (`device_class: problem`) and
  the raw alarm bitmasks A/B/C (diagnostic).
- **Solar thermal:** added pump, status and collector/tank temperatures (only when
  a solar function is configured).
- **Energy:** added the import/export high-word diagnostic registers
  (`EM*_ACCHI` / `EM4_ACC2HI`); combine with the low word as
  `(ACCHI*65536 + ACCLO)/10` in a Home Assistant template sensor for totals beyond
  6553.5 kWh.

## 1.2.1

- Fix: saving the configuration failed with `Missing required option 'mqtt_host'`.
  The optional `mqtt_host` / `mqtt_port` / `mqtt_user` / `mqtt_password` options had
  a `null` default in `options:`, which Home Assistant rejects when the field is left
  empty. They are now declared only in `schema:` (optional), so leaving them unset is
  valid. To use a manual external broker, set them via the add-on's "Edit in YAML".

## 1.2.0

Merges upstream **1.1.2** (network capability for HA Supervised / Debian 12,
`MTx_MODE` sensor + `MTx_FORCING` selector, revised zone-active logic, climate
command-template decimals fix, Alpine 3.23 base image, armv7 dropped) and adds:

- Fix `ERROR: Got unexpected response from the API: Service not enabled` crash at
  startup: the start script now guards the Supervisor MQTT lookup with
  `bashio::services.available` and exits with a clear, actionable message instead
  of crashing when no MQTT provider (e.g. the Mosquitto broker add-on) is published.
- Add optional `mqtt_host` / `mqtt_port` / `mqtt_user` / `mqtt_password` options so
  the add-on can use a manually configured external broker when no Supervisor MQTT
  service exists; manual settings take precedence over auto-discovery.
- Honour the broker port (Supervisor-provided or manual) instead of hardcoding 1883.
- Change the MQTT service dependency from `need` to `want`.
- Fix: fan flow-rate setpoints (`D*_SPEED_*`) were written ~100× too large because
  the per-parameter `command_template` was ignored.
- Fix: humidity-less zone `climate` entities had a non-unique `unique_id` that could
  collide across multiple stations.
- Entities are no longer all marked `diagnostic`: real measurements (temperatures,
  humidity, power, energy, operational states) are now primary entities, and
  `climate` is no longer demoted to `config`. Only alarms, generic digital inputs,
  tuning/hysteresis values and the last-update timestamp stay diagnostic.
- Entities now report availability via an MQTT Last-Will/birth on
  `setecna/<systemID>/status`, so they show as **unavailable** when the add-on stops
  or crashes.
- MQTT discovery (`.../config`) messages **and entity state** are published
  **retained**, so after a Home Assistant or broker restart entities keep their last
  value instead of showing as *unknown* until the add-on is restarted.
- Device card enriched with a `configuration_url` deep-link to the Setecna web UI.
- New optional `debug_dump` option writes the full raw `getres` payload and a list
  of not-yet-mapped parameter IDs to `/share` once at startup, to help discover
  additional data the station exposes. Default off.

## 1.1.2

- Bump home-assistant/builder from 2024.08.2 to 2025.03.0
- Bump docker/login-action from 3.3.0 to 3.4.0
- Bump docker/login-action from 3.4.0 to 3.7.0
- Bump actions/checkout from 4.2.2 to 6.0.2
- Bump home-assistant/builder from 2025.03.0 to 2025.11.0
- Bump frenck/action-addon-linter from 2.18 to 2.21

## 1.1.1

- Bump home-assistant/builder from 2024.08.1 to 2024.08.2
- Bump actions/checkout from 4.1.7 to 4.2.1
- Bump frenck/action-addon-linter from 2.15 to 2.17
- Bump actions/checkout from 4.2.1 to 4.2.2
- Bump frenck/action-addon-linter from 2.17 to 2.18
- Add network capability to make this run on HA Supervised on Debian 12

## 1.1.0

- Add MTx_MODE as a sensor and MTx_FORCING as a selector to HomeAssistant
- **BREAKING CHANGE**: Change how a zone is considered active, now the plugin check if Zx_SENSOR_CHN != 0 instead of Zx_TEMP != 32769 (aligned with the web interface logic)

## 1.0.1

- Fixes decimals in command template for climate entities

## 1.0.0

- Initial release
