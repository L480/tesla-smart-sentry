# <img style="display: inline; height: 1em; width: auto;" src="images/sentry-mode-icon.png" alt="logo" /> tesla-smart-sentry</img>

![GitHub release (with filter)](https://img.shields.io/github/v/release/L480/tesla-smart-sentry)

Enables Tesla Sentry Mode via Zigbee motion sensors.

![Shell executing tesla-smart-sentry](/images/shell.png)

💾 [Downloads](#downloads)

## What is Sentry Mode?

Tesla Sentry Mode records suspicious activities around the vehicle when it's locked and in Park.

Since Sentry Mode requires the vehicle to remain awake, the vehicle's idle consumption with enabled Sentry Mode is between 150 and 300 watts. To save energy tesla-smart-sentry uses Zigbee motion sensors and the Philips Hue Bridge to enabled Sentry Mode on-demand based on motion.

## Hardware Requirements

- [Philips Hue Bridge](https://www.philips-hue.com/en-us/p/hue-bridge/046677458478#overview)
- [Philips Hue outdoor sensor](https://www.philips-hue.com/en-us/p/hue-outdoor-sensor/046677570989#overview)
  - Original Philips Hue sensors are recommended as you can configure sensor sensitivity.
- [Raspberry Zero](https://www.raspberrypi.com/products/raspberry-pi-zero-2-w/)
  - Or any other ARM/ARM64/amd64 Linux board.
  - BCM43* series chips are notorious for problems when both Wi-Fi and Bluetooth are used at the same time. An external Bluetooth dongle may be required.

## Set up tesla-smart-sentry

You can operate tesla-smart-sentry in three modes:

| Mode    | Description                                                                                                                                                                          | Sentry Mode Activation Time                                   |
| ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------- |
| API     | Sends an end-to-end authenticated command via the Tesla Fleet API to the vehicle when motion is detected.<br>*Raspberry Pi needs access to the Hue Bridge and the internet.*         | Sleep: 15-20 secs<br>Awake: 1-3 secs                          |
| Webhook | Sends a webhook to any HTTP endpoint when motion is detected.<br>*Raspberry Pi needs access to the Hue Bridge and the webhook endpoint.*                                             | Depends on webhook endpoint                                   |
| BLE     | Sends an end-to-end authenticated command via Bluetooth directly to the vehicle.<br>*Raspberry Pi needs access to the Hue Bridge and must be within Bluetooth range of the vehicle.* | Sleep: 3-15 secs<br>Awake: 3-15 secs<br>Depends on distance |

In addition to tesla-smart-sentry, there is also tesla-ble-proxy, a Bluetooth proxy provided as part of this repository that can be used in conjunction with webhook mode. A reference architecture can be found [here](./tesla-ble-proxy.md).

### API Mode

> [!NOTE]
> Enabling Sentry Mode takes between 15 and 20 seconds when the vechicle is asleep and is almost instantaneous when the vehicle is awake (same behaviour as the Tesla app when not in Bluetooth range of the vehicle).

1. Set up a Tesla Developer account, register an application, create a key-pair, and get the OAuth refresh token of your current user as describe [here](https://developer.tesla.com/docs/fleet-api#setup).
2. Create a new user on your Hue Bridge as described [here](https://developers.meethue.com/develop/get-started-2/).
3. Get the Hue sensor IDs through `https://<HUE BRIDGE IP>/api/<HUE TOKEN FROM PREVIOUS STEP>/sensors` (JSON object names representing the sensor IDs).
4. Run tesla-smart-sentry in Docker:

    ```bash
    # Create a directory for your tokens and private key
    mkdir /opt/tesla-smart-sentry
    # Copy your private key
    cp private-key.pem /opt/tesla-smart-sentry
    # Make the directory accessible from the tesla-smart-sentry container
    chown -R 10000:10000 /opt/tesla-smart-sentry
    # Run tesla-smart-sentry
    docker run -d \
      -e MODE='api' \
      -e HUE_BRIDGE_SSE_URL='https://hue-bridge/eventstream/clip/v2' \
      -e HUE_BRIDGE_TOKEN='9V63a7kL9zynL9V63a7kL9zynLZgC98xFq2x8ZgC98xFq2x8' \
      -e HUE_SENSORS='/sensors/1,/sensors/4' \
      -e TESLA_VIN='5YJSA1E62NF016329' \
      -e TESLA_PRIVATE_KEY_FILE='/var/lib/auth/private-key.pem' \
      -e TESLA_REFRESH_TOKEN='9V63a7kL9zynguMM9vyZNqVmCKcLPbtENpy4LZgC98xFq2x8' \
      -e TESLA_CLIENT_ID='e27346f1-eca0-4b66-b2be-676c61a728d8' \
      -e TESLA_SENTRY_TIMEOUT='10' \
      -e ARMED_SCHEDULE='00:00-24:00' \
      -v /opt/tesla-smart-sentry:/var/lib/auth \
      -v /etc/localtime:/etc/localtime:ro \
      --name tesla-smart-sentry \
      ghcr.io/l480/tesla-smart-sentry:latest
    ```

#### Environment Variables

| Environment Variable       | Description                                                                           | Example                                            |
| -------------------------- | ------------------------------------------------------------------------------------- | -------------------------------------------------- |
| **MODE**                   | tesla-smart-sentry operating mode                                                     | `api`                                              |
| **HUE_BRIDGE_SSE_URL**     | Server-Sent Events (SSE) endpoint of the Hue Bridge                                   | `https://192.168.1.10/eventstream/clip/v2`         |
| **HUE_BRIDGE_TOKEN**       | Credentials for the Hue Bridge                                                        | `9V63a7kL9zynL9V63a7kL9zynLZgC98xFq2x8ZgC98xFq2x8` |
| **HUE_SENSORS**            | List of Hue sensors which should be used for motion detection                         | `/sensors/1,/sensors/4`                            |
| **TESLA_VIN**              | Vehicle identification number                                                         | `5YJSA1E62NF016329`                                |
| **TESLA_PRIVATE_KEY_FILE** | A PEM-encoded EC private key using the secp256r1 curve (prime256v1)                   | `/var/lib/auth/private-key.pem`                    |
| **TESLA_REFRESH_TOKEN**    | OAuth refresh token of Tesla's "Third-party token"                                    | `9V63a7kL9zynguMM9vyZNqVmCKcLPbtENpy4LZgC98xFq2x8` |
| **TESLA_CLIENT_ID**        | Application's client ID                                                               | `e27346f1-eca0-4b66-b2be-676c61a728d8`             |
| **TESLA_SENTRY_TIMEOUT**   | Time in minutes after Sentry Mode is disabled when no motion is detected              | `10`                                               |
| **ARMED_SCHEDULE**         | Time period for Sentry Mode to activate when motion is detected (format: HH:MM-HH:MM) | `00:00-24:00`                                      |

### Webhook Mode

> [!NOTE]
> The activation of Sentry Mode should be outsourced to the service which receives the webhook. [tesla-ble-proxy](./tesla-ble-proxy.md) can be used for this.

1. Create a new user on your Hue Bridge as described [here](https://developers.meethue.com/develop/get-started-2/).
2. Get the Hue sensor IDs through `https://<HUE BRIDGE IP>/api/<HUE TOKEN FROM PREVIOUS STEP>/sensors` (JSON object names representing the sensor IDs).
3. Run tesla-smart-sentry in Docker:

    ```bash
    # Run tesla-smart-sentry
    docker run -d \
      -e MODE='webhook' \
      -e HUE_BRIDGE_SSE_URL='https://hue-bridge/eventstream/clip/v2' \
      -e HUE_BRIDGE_TOKEN='9V63a7kL9zynL9V63a7kL9zynLZgC98xFq2x8ZgC98xFq2x8' \
      -e HUE_SENSORS='/sensors/1,/sensors/4' \
      -e TESLA_SENTRY_TIMEOUT='10' \
      -e ARMED_SCHEDULE='00:00-24:00' \
      -e WEBHOOK_ON_URL='https://example.com/on' \
      -e WEBHOOK_ON_BODY='{"secret": "qQysb56jxyekrKTmkturtAMw"}' \
      -e WEBHOOK_OFF_URL='https://example.com/off' \
      -e WEBHOOK_OFF_BODY='{"secret": "qQysb56jxyekrKTmkturtAMw"}' \
      -e WEBHOOK_TLS_VERIFY='true' \
      -v /etc/localtime:/etc/localtime:ro \
      --name tesla-smart-sentry \
      ghcr.io/l480/tesla-smart-sentry:latest
    ```

#### Environment Variables

| Environment Variable     | Description                                                                           | Example                                            |
| ------------------------ | ------------------------------------------------------------------------------------- | -------------------------------------------------- |
| **MODE**                 | tesla-smart-sentry operating mode                                                     | `api`                                              |
| **HUE_BRIDGE_SSE_URL**   | Server-Sent Events (SSE) endpoint of the Hue Bridge                                   | `https://192.168.1.10/eventstream/clip/v2`         |
| **HUE_BRIDGE_TOKEN**     | Credentials for the Hue Bridge                                                        | `9V63a7kL9zynL9V63a7kL9zynLZgC98xFq2x8ZgC98xFq2x8` |
| **HUE_SENSORS**          | List of Hue sensors which should be used for motion detection                         | `/sensors/1,/sensors/4`                            |
| **TESLA_SENTRY_TIMEOUT** | Time in minutes after Sentry mode is disabled when no motion is detected              | `10`                                               |
| **ARMED_SCHEDULE**       | Time period for Sentry mode to activate when motion is detected (format: HH:MM-HH:MM) | `00:00-24:00`                                      |
| **WEBHOOK_ON_URL**       | HTTP endpoint URL to trigger when motion is detected                                  | `https://example.com/on`                           |
| **WEBHOOK_ON_BODY**      | HTTP body of **WEBHOOK_ON_URL** request                                               | `{"secret": "qQysb56jxyekrKTmkturtAMw"}`           |
| **WEBHOOK_OFF_URL**      | HTTP endpoint URL to trigger when **TESLA_SENTRY_TIMEOUT** is reached                 | `https://example.com/off`                          |
| **WEBHOOK_OFF_BODY**     | HTTP body of **WEBHOOK_ON_URL** request                                               | `{"secret": "qQysb56jxyekrKTmkturtAMw"}`           |
| **WEBHOOK_TLS_VERIFY**   | Verify TLS certificate                                                                | `true`                                             |

### BLE Mode

> [!NOTE]  
> Enabling Sentry Mode via Bluetooth is faster than via API. However, you must be within Bluetooth range of the vehicle. This mode does not work on Windows OS.

> [!WARNING]  
> Make sure to protect your private key when placing a Raspberry Pi outside near your vehicle. If your Raspberry Pi is stolen, your private key is at risk and could be used to access your vehicle. Check out the [tesla-ble-proxy reference architecture](./tesla-ble-proxy.md).

1. Create a key-pair and enroll the public key to your vehicle as described [here](https://github.com/teslamotors/vehicle-command/tree/main/cmd/tesla-control#key-management).
2. Create a new user on your Hue Bridge as described [here](https://developers.meethue.com/develop/get-started-2/).
3. Get the Hue sensor IDs through `https://<HUE BRIDGE IP>/api/<HUE TOKEN FROM PREVIOUS STEP>/sensors` (JSON object names representing the sensor IDs).
4. Run tesla-smart-sentry locally:

    ```bash
    # Set your architecture
    ARCH=arm64
    # Copy your private key
    mkdir /opt/tesla-smart-sentry
    cp private-key.pem /opt/tesla-smart-sentry
    # Install binary
    wget https://github.com/L480/tesla-smart-sentry/releases/latest/download/tesla-smart-sentry-linux-$ARCH
    sudo chmod +x tesla-smart-sentry-linux-$ARCH
    sudo mv tesla-smart-sentry-linux-$ARCH /usr/local/bin/tesla-smart-sentry
    # Set up service
    sudo sh -c 'echo "[Unit]
    Description=tesla-smart-sentry
    After=network-online.target

    [Service]
    Type=idle
    ExecStart=/usr/local/bin/tesla-smart-sentry
    Restart=always
    RestartSec=3
    StandardOutput=syslog
    StandardError=syslog
    SyslogIdentifier=tesla-smart-sentry

    [Install]
    WantedBy=multi-user.target" > /etc/systemd/system/tesla-smart-sentry.service'
    sudo systemctl daemon-reload
    # Add/overwrite your environment variables as described here: https://serverfault.com/a/413408
    sudo systemctl edit tesla-smart-sentry
    # Enable autostart and start service
    sudo systemctl enable tesla-smart-sentry
    sudo systemctl start tesla-smart-sentry
    ```

#### Environment Variables

| Environment Variable       | Description                                                                           | Example                                            |
| -------------------------- | ------------------------------------------------------------------------------------- | -------------------------------------------------- |
| **MODE**                   | tesla-smart-sentry operating mode                                                     | `api`                                              |
| **HUE_BRIDGE_SSE_URL**     | Server-Sent Events (SSE) endpoint of the Hue Bridge                                   | `https://192.168.1.10/eventstream/clip/v2`         |
| **HUE_BRIDGE_TOKEN**       | Credentials for the Hue Bridge                                                        | `9V63a7kL9zynL9V63a7kL9zynLZgC98xFq2x8ZgC98xFq2x8` |
| **HUE_SENSORS**            | List of Hue sensors which should be used for motion detection                         | `/sensors/1,/sensors/4`                            |
| **TESLA_VIN**              | Vehicle identification number                                                         | `5YJSA1E62NF016329`                                |
| **TESLA_PRIVATE_KEY_FILE** | A PEM-encoded EC private key using the secp256r1 curve (prime256v1)                   | `/var/lib/auth/private-key.pem`                    |
| **TESLA_SENTRY_TIMEOUT**   | Time in minutes after Sentry Mode is disabled when no motion is detected              | `10`                                               |
| **ARMED_SCHEDULE**         | Time period for Sentry Mode to activate when motion is detected (format: HH:MM-HH:MM) | `00:00-24:00`                                      |

## Downloads

- **tesla-smart-sentry**
  - `docker pull ghcr.io/l480/tesla-smart-sentry:latest`
  - [linux-arm64](https://github.com/L480/tesla-smart-sentry/releases/latest/download/tesla-smart-sentry-linux-arm64)
  - [linux-arm](https://github.com/L480/tesla-smart-sentry/releases/latest/download/tesla-smart-sentry-linux-arm)
  - [linux-amd64](https://github.com/L480/tesla-smart-sentry/releases/latest/download/tesla-smart-sentry-linux-amd64)
- **tesla-ble-proxy**
  - `docker pull ghcr.io/l480/tesla-ble-proxy:latest`
  - [linux-arm64](https://github.com/L480/tesla-smart-sentry/releases/latest/download/tesla-ble-proxy-linux-arm64)
  - [linux-arm](https://github.com/L480/tesla-smart-sentry/releases/latest/download/tesla-ble-proxy-linux-arm)
  - [linux-amd64](https://github.com/L480/tesla-smart-sentry/releases/latest/download/tesla-ble-proxy-linux-amd64)
