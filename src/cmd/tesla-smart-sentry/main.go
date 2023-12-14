package main

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/L480/tesla-smart-sentry/internal/hue"
	"github.com/L480/tesla-smart-sentry/internal/logger"
	"github.com/L480/tesla-smart-sentry/internal/request"
	"github.com/L480/tesla-smart-sentry/internal/tesla"
	"github.com/L480/tesla-smart-sentry/internal/util"
	"github.com/teslamotors/vehicle-command/pkg/vehicle"
)

var Env = map[string]string{
	"EnvMode":                "MODE",
	"EnvHueBridgeSseUrl":     "HUE_BRIDGE_SSE_URL",
	"EnvHueBridgeToken":      "HUE_BRIDGE_TOKEN",
	"EnvHueSensors":          "HUE_SENSORS",
	"EnvTeslaVin":            "TESLA_VIN",
	"EnvTeslaPrivateKeyFile": "TESLA_PRIVATE_KEY_FILE",
	"EnvTeslaRefreshToken":   "TESLA_REFRESH_TOKEN",
	"EnvTeslaClientId":       "TESLA_CLIENT_ID",
	"EnvTeslaSentryTimeout":  "TESLA_SENTRY_TIMEOUT",
	"EnvArmedSchedule":       "ARMED_SCHEDULE",
	"EnvWebhookOnUrl":        "WEBHOOK_ON_URL",
	"EnvWebhookOnBody":       "WEBHOOK_ON_BODY",
	"EnvWebhookOffUrl":       "WEBHOOK_OFF_URL",
	"EnvWebhookOffBody":      "WEBHOOK_OFF_BODY",
	"EnvWebhookTlsVerify":    "WEBHOOK_TLS_VERIFY",
}

var sentryTimer *time.Timer = time.NewTimer(0)
var contextTimeout time.Duration = 30 * time.Second

func sentryTimeout(c func(bool)) {
	<-sentryTimer.C
	sentryTimer.Stop()
	for {
		<-sentryTimer.C
		c(false)
	}
}

func main() {
	logger.SetLevel(logger.LevelDebug)
	mode := strings.ToLower(os.Getenv(Env["EnvMode"]))
	hueBridgeSseUrl := os.Getenv(Env["EnvHueBridgeSseUrl"])
	hueBridgeToken := os.Getenv(Env["EnvHueBridgeToken"])
	hueSensors := strings.Split(os.Getenv(Env["EnvHueSensors"]), ",")
	teslaVin := os.Getenv(Env["EnvTeslaVin"])
	teslaPrivateKeyFile := os.Getenv(Env["EnvTeslaPrivateKeyFile"])
	teslaRefreshToken := os.Getenv(Env["EnvTeslaRefreshToken"])
	teslaClientId := os.Getenv(Env["EnvTeslaClientId"])
	i, _ := strconv.Atoi(os.Getenv(Env["EnvTeslaSentryTimeout"]))
	teslaSentryTimeout := time.Duration(i) * time.Minute
	armedSchedule := os.Getenv(Env["EnvArmedSchedule"])
	webhookOnUrl := os.Getenv(Env["EnvWebhookOnUrl"])
	webhookOnBody := os.Getenv(Env["EnvWebhookOnBody"])
	webhookOffUrl := os.Getenv(Env["EnvWebhookOffUrl"])
	webhookOffBody := os.Getenv(Env["EnvWebhookOffBody"])
	webhookTlsVerify, _ := strconv.ParseBool(os.Getenv(Env["EnvWebhookTlsVerify"]))

	ble := false
	sentryState := false

	if mode == "api" && util.CheckEnvs([]string{
		Env["EnvHueBridgeSseUrl"],
		Env["EnvHueBridgeToken"],
		Env["EnvHueSensors"],
		Env["EnvTeslaVin"],
		Env["EnvTeslaPrivateKeyFile"],
		Env["EnvTeslaRefreshToken"],
		Env["EnvTeslaClientId"],
		Env["EnvTeslaSentryTimeout"],
		Env["EnvArmedSchedule"]}) {
	} else if mode == "webhook" && util.CheckEnvs([]string{
		Env["EnvHueBridgeSseUrl"],
		Env["EnvHueBridgeToken"],
		Env["EnvHueSensors"],
		Env["EnvTeslaSentryTimeout"],
		Env["EnvArmedSchedule"],
		Env["EnvWebhookOnUrl"],
		Env["EnvWebhookOnBody"],
		Env["EnvWebhookOffUrl"],
		Env["EnvWebhookOffBody"],
		Env["EnvWebhookTlsVerify"]}) {
	} else if mode == "ble" && util.CheckEnvs([]string{
		Env["EnvHueBridgeSseUrl"],
		Env["EnvHueBridgeToken"],
		Env["EnvHueSensors"],
		Env["EnvTeslaVin"],
		Env["EnvTeslaPrivateKeyFile"],
		Env["EnvTeslaSentryTimeout"],
		Env["EnvArmedSchedule"]}) {
		ble = true
	} else {
		logger.Error("Environment variables missing.")
		os.Exit(1)
	}

	logger.Info("🚀 TESLA-SMART-SENTRY")
	logger.Info("\033[1mMode:\033[0m %s | \033[1mMonitored Sensors:\033[0m %s | \033[1mSchedule:\033[0m %sh", mode, hueSensors, armedSchedule)

	teslaConfig := tesla.Config{
		Vin:              teslaVin,
		PrivateKeyFile:   teslaPrivateKeyFile,
		Ble:              ble,
		AccessTokenFile:  "/var/lib/auth/access-token",
		RefreshTokenFile: "/var/lib/auth/refresh-token",
		ClientId:         teslaClientId,
		RefreshToken:     teslaRefreshToken,
	}

	go sentryTimeout(func(c bool) {
		logger.Info("No more motion has been detected for %s minutes. Disabling Sentry Mode...", strconv.FormatFloat((teslaSentryTimeout.Minutes()), 'g', 2, 64))
		if mode == "webhook" {
			webhookEndpoint := request.Endpoint{
				Url:                webhookOffUrl,
				Method:             "POST",
				Headers:            nil,
				Body:               webhookOffBody,
				InsecureSkipVerify: webhookTlsVerify,
			}
			resp, _ := request.Connect(webhookEndpoint)
			if resp.StatusCode != 200 {
				logger.Error("Failed to disable Sentry Mode (received HTTP %d).", resp.StatusCode)
			} else {
				sentryState = false
			}
			resp.Body.Close()
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
			defer cancel()
			if err := tesla.Execute(ctx, teslaConfig, false, func(car *vehicle.Vehicle) error {
				return car.SetSentryMode(ctx, false)
			}); err != nil {
				logger.Error("Failed to disable Sentry Mode: %s", err)
			} else {
				sentryState = false
			}
		}

		if !sentryState {
			logger.Info("\033[32mSentry Mode is disabled.\033[0m")
		}
	})

	if mode == "api" {
		go tesla.RefreshToken(teslaConfig)
	}

	hueBridge := request.Endpoint{
		Url:                hueBridgeSseUrl,
		Method:             "GET",
		Headers:            map[string]string{"hue-application-key": hueBridgeToken, "Accept": "text/event-stream"},
		Body:               "",
		InsecureSkipVerify: true,
	}
	resp, err := request.Connect(hueBridge)
	if err != nil {
		logger.Error("Failed to connect to Hue bridge: %s", err)
		os.Exit(1)
	}

	sse := request.SubscribeSse(resp, func(c []byte) {
		motion := hue.CheckMotion(c, hueSensors)
		armed := util.InTimeframe(armedSchedule)
		if armed && motion && !sentryState {
			logger.Info("Motion deteced. Enabling Sentry Mode...")
			if mode == "webhook" {
				webhookEndpoint := request.Endpoint{
					Url:                webhookOnUrl,
					Method:             "POST",
					Headers:            nil,
					Body:               webhookOnBody,
					InsecureSkipVerify: webhookTlsVerify,
				}
				resp, _ := request.Connect(webhookEndpoint)
				if resp.StatusCode != 200 {
					logger.Error("Failed to enable Sentry Mode (received HTTP %d).", resp.StatusCode)
				} else {
					sentryState = true
				}
				resp.Body.Close()
			} else {
				if mode == "ble" {
					ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
					defer cancel()
					tesla.Execute(ctx, teslaConfig, true, func(car *vehicle.Vehicle) error {
						return car.Wakeup(ctx)
					})
				}
				ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
				defer cancel()
				if err := tesla.Execute(ctx, teslaConfig, false, func(car *vehicle.Vehicle) error {
					return car.SetSentryMode(ctx, true)
				}); err != nil {
					logger.Error("Failed to enable Sentry Mode: %s", err)
				} else {
					sentryState = true
				}
			}
			if sentryState {
				logger.Info("\033[31mSentry Mode is enabled.\033[0m")
				sentryTimer.Reset(teslaSentryTimeout)
				logger.Info("Sentry mode ends %s minutes after no more motion is detected.", strconv.FormatFloat((teslaSentryTimeout.Minutes()), 'g', 2, 64))
			}
		} else if armed && motion && sentryState {
			sentryTimer.Reset(teslaSentryTimeout)
		}
	})
	if sse != nil {
		logger.Error("Failed to subscribe to Hue bridge: %s", sse)
		os.Exit(1)
	}
}
