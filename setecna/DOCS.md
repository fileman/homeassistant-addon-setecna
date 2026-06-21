# DISCLAIMER

This add-on is developed by reverse engineering the Setecna web-interface and is not officially supported by the Setecna team, use it with caution.

# Home Assistant Add-on: Setecna add-on

## Why this addon

This add-on is meant to integrate your REG based thermal power plant in Home Assistant.

It's a web based integration, so your system needs to have access to the internet in order to comunicate with Setecna servers first.

## Prerequisites

Before you can use this add-on you need to:
1. Install an MQTT add-on
1. Enable and configure the MQTT integration

*The Setecna add-on will automatically connect to the MQTT broker installed in your Home Assistance instance.*

## How to use


Once installed use the "configuration" tab to insert the following informations:

### Required parameters
- SystemID ( find it once logged to the Setecna web-interface )
- Username
- Password

### Optional parameters
- Readonly:
    - `OFF` = All parameters will be created in HA as `sensors` or `binary_sensors`, disabling any possible interactions beetween Home Assistant and your REG system.
    - `ON` = Configuration parameters that you can change from Setecna web-interface will be created as `numbers` inside Home Assistant, enabling you to control your system inside Home Assistant
- Advanced integration:
    - `ON` = The addon will match Home Assistant's built-in entities such as `climate` or `water_heater` with REG systems's zones and DWH parameters.
    - `OFF` = Home Assistant's built-in entities will not be created by the add-on (advanced users can still create them in their homeassistant's configuration.yaml)

### MQTT broker (manual)

By default the add-on auto-detects the MQTT broker from the **Mosquitto broker** add-on. If you use an external broker that is not published as a Home Assistant service (e.g. a standalone broker, or one configured only through the MQTT *integration*), set these so the add-on can connect:

- `mqtt_host`, `mqtt_port`, `mqtt_user`, `mqtt_password`

Leave them empty to use auto-detection. When `mqtt_host` is set, the manual values take precedence over auto-detection.

### Debugging / discovering more data

- `debug_dump`:
    - `ON` = on the next start, the add-on writes two files to your Home Assistant `/share` folder: `setecna_getres_dump.json` (the full raw data the station returns) and `setecna_unmapped_ids.json` (parameters the add-on does **not** currently expose as entities). These help identify additional data that could be integrated. **Turn it back OFF after capturing** — the dump contains your station telemetry and `systemID`, so review/redact before sharing it publicly.
    - `OFF` = no debug files are written (default).