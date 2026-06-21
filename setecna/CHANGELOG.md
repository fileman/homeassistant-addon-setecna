<!-- https://developers.home-assistant.io/docs/add-ons/presentation#keeping-a-changelog -->

## 1.1.2

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
- MQTT discovery (`.../config`) messages are published **retained**, so entities
  survive a Home Assistant or broker restart.
- Device card enriched with a `configuration_url` deep-link to the Setecna web UI.
- New optional `debug_dump` option writes the full raw `getres` payload and a list
  of not-yet-mapped parameter IDs to `/share` once at startup, to help discover
  additional data the station exposes. Default off.
- Bump the base image off long-EOL Alpine 3.15 to 3.22.

## 1.0.0

- Initial release
