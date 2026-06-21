<!-- https://developers.home-assistant.io/docs/add-ons/presentation#keeping-a-changelog -->

## 1.0.1

- Fix `ERROR: Got unexpected response from the API: Service not enabled` crash at
  startup: the start script now guards the Supervisor MQTT lookup with
  `bashio::services.available` instead of calling `bashio::services mqtt`
  unconditionally. When no MQTT provider is published, the add-on now exits with a
  clear, actionable message instead of crashing.
- Add optional `mqtt_host` / `mqtt_port` / `mqtt_user` / `mqtt_password` options so
  the add-on can connect to a manually configured (e.g. external) MQTT broker when
  the Supervisor MQTT service is unavailable. Manual settings take precedence over
  Supervisor auto-discovery.
- Honour the broker port (Supervisor-provided or manual) instead of hardcoding 1883.
- Change the MQTT service dependency from `need` to `want` (a manual broker is now
  a supported alternative to a provider add-on).
- Bump the base image from the long-EOL Alpine 3.15 to 3.22.

## 1.0.0

- Initial release
