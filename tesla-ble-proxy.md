# tesla-ble-proxy

The purpose of tesla-ble-proxy is to enable Sentry Mode via Bluetooth without putting your private key at risk when the Raspberry Pi is operating in a physically unsafe environment (like your backyard).

![tesla-ble-proxy architectiure](./images/tesla-ble-proxy.png)

1. tesla-smart-sentry (which holds your private key), operating in a secure environment (e.g. inside a building), subscribes to the Hue Bridge and checks for motion.
2. Zigbee motion sensor detects motion.
3. Status change is being sent to Hue Bridge via the Zigbee protocol.
4. teala-sentry-mode recognizes the motion and triggers a webhook to tesla-ble-proxy with the private key in the HTTP request body.
5. tesla-ble-proxy sents an end-to-end authenticated Bluetooth command to your vehicle.

The private key only lives in the Raspberry Pi's RAM.

## Set up tesla-ble-proxy

1. Create a key-pair and enroll the public key to your vehicle as described [here](https://github.com/teslamotors/vehicle-command/tree/main/cmd/tesla-control#key-management).
2. Run tesla-smart-sentry locally:

    ```bash
    # Set your architecture
    ARCH=arm64
    # Install binary
    wget https://github.com/L480/tesla-smart-sentry/releases/latest/download/tesla-ble-proxy-linux-$ARCH
    sudo chmod +x tesla-ble-proxy-linux-$ARCH
    sudo mv tesla-ble-proxy-linux-$ARCH /usr/local/bin/tesla-ble-proxy
    # Set up service
    sudo sh -c 'echo "[Unit]
    Description=tesla-ble-proxy
    After=network-online.target

    [Service]
    Type=idle
    ExecStart=/usr/local/bin/tesla-ble-proxy
    Restart=always
    RestartSec=3
    StandardOutput=syslog
    StandardError=syslog
    SyslogIdentifier=tesla-ble-proxy

    [Install]
    WantedBy=multi-user.target" > /etc/systemd/system/tesla-ble-proxy.service'
    sudo systemctl daemon-reload
    # Enable autostart and start service
    sudo systemctl enable tesla-ble-proxy
    sudo systemctl start tesla-ble-proxy
    ```

## Send commands

The below JSON schema can be used to send POST requests against tesla-ble-proxy.

> [!NOTE]  
> Make sure to replace line breaks in the private key with `\n`.

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "tesla-ble-proxy Schema",
  "type": "object",
  "properties": {
    "vin": {
      "type": "string"
    },
    "private_key": {
      "type": "string"
    }
  },
  "required": [
    "vin",
    "private_key"
  ]
}
```

### Sentry Mode

Turning on Sentry Mode:

```bash
curl -H 'Content-Type: application/json' \
  -d '{"vin":"5YJSA1E62NF016329","private_key":"-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIC6CbhAGehnSMfFUpLX80YbuulbRY5PbMgO3E52xnJ48oAoGCCqGSM49\nAwEHoUQDQgAEptaF+LYs0uFRPqwpbZB/7SQ8TMFwblHOP5VoOAzs7QDnJFcAn1kS\nC5WkXCPzsYluBvUAuCT5ObfYHYNQ26uuEQ==\n-----END EC PRIVATE KEY-----"}' \
  -X POST \
  http://tesla-ble-proxy:8080/sentry-mode/on
```

Turning off Sentry Mode:

```bash
curl -H 'Content-Type: application/json' \
  -d '{"vin":"5YJSA1E62NF016329","private_key":"-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIC6CbhAGehnSMfFUpLX80YbuulbRY5PbMgO3E52xnJ48oAoGCCqGSM49\nAwEHoUQDQgAEptaF+LYs0uFRPqwpbZB/7SQ8TMFwblHOP5VoOAzs7QDnJFcAn1kS\nC5WkXCPzsYluBvUAuCT5ObfYHYNQ26uuEQ==\n-----END EC PRIVATE KEY-----"}' \
  -X POST \
  http://tesla-ble-proxy:8080/sentry-mode/off
```

### Health Check

```bash
curl -I http://tesla-ble-proxy:8080/health
```
