package tesla

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/teslamotors/vehicle-command/pkg/account"
	"github.com/teslamotors/vehicle-command/pkg/connector/ble"
	"github.com/teslamotors/vehicle-command/pkg/protocol"
	"github.com/teslamotors/vehicle-command/pkg/vehicle"
)

type Config struct {
	Vin            string
	PrivateKeyFile string
	Ble            bool

	// Only required when Ble is false.
	AccessTokenFile  string
	RefreshTokenFile string
	ClientId         string
	RefreshToken     string
}

func validateResult(err error) error {
	if protocol.MayHaveSucceeded(err) {
		return fmt.Errorf("command sent, but client could not confirm receipt: %s", err)
	} else {
		return fmt.Errorf("failed to execute command: %s", err)
	}
}

func Execute(c Config, vcsecOnly bool, cmd func(context.Context, *vehicle.Vehicle) error) error {
	var timeout time.Duration
	if c.Ble {
		timeout = 15 * time.Second
	} else {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var car *vehicle.Vehicle
	privateKey, err := protocol.LoadPrivateKey(c.PrivateKeyFile)
	if err != nil {
		return fmt.Errorf("failed to load private key: %s", err)
	}

	if c.Ble {
		conn, err := ble.NewConnection(ctx, c.Vin)
		if err != nil {
			return fmt.Errorf("failed to connect to vehicle: %s", err)
		}

		c, err := vehicle.NewVehicle(conn, privateKey, nil)
		if err != nil {
			return fmt.Errorf("failed to connect to vehicle: %s", err)
		}
		car = c
	} else {
		accessToken, err := os.ReadFile(c.AccessTokenFile)
		if err != nil {
			return fmt.Errorf("failed to load access token: %s", err)
		}

		acct, err := account.New(string(accessToken), "")
		if err != nil {
			return fmt.Errorf("authentication error: %s", err)
		}

		c, err := acct.GetVehicle(ctx, c.Vin, privateKey, nil)
		if err != nil {
			return fmt.Errorf("failed to fetch vehicle info from account: %s", err)
		}
		car = c
	}

	if err := car.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to vehicle: %s", err)
	}
	defer car.Disconnect()

	if c.Ble {
		retries := 5 // Retry handshake and command execution up to x times to workaround race conditions with BLE
		var domains []protocol.Domain
		if vcsecOnly {
			domains = append(domains, protocol.DomainVCSEC)
		} else {
			domains = append(domains, protocol.DomainVCSEC)
			domains = append(domains, protocol.DomainInfotainment)
		}
		for i := 1; i <= retries; i++ {
			sessionCtx, sessionCancel := context.WithTimeout(context.Background(), 2*time.Second)
			if err := car.StartSession(sessionCtx, domains); err != nil {
				if i != retries {
					sessionCancel()
					continue
				} else {
					sessionCancel()
					return fmt.Errorf("failed to perform handshake with vehicle: %s", err)
				}
			}
			sessionCancel()
		}
		for i := 1; i <= retries; i++ {
			cmdCtx, cmdCancel := context.WithTimeout(context.Background(), 2*time.Second)
			if err := cmd(cmdCtx, car); err != nil {
				if i != retries {
					cmdCancel()
					continue
				} else {
					cmdCancel()
					return fmt.Errorf("failed to execute command: %s", err)
				}
			}
			cmdCancel()
		}
	} else {
		if err := car.Wakeup(ctx); err != nil {
			return fmt.Errorf("failed to wake up vehicle: %s", err)
		}
		if err := car.StartSession(ctx, nil); err != nil {
			return fmt.Errorf("failed to perform handshake with vehicle: %s", err)
		}
		for {
			if err := car.Ping(ctx); err != nil {
				time.Sleep(1 * time.Second)
				continue
			}
			break
		}
		if err := cmd(ctx, car); err != nil {
			return validateResult(err)
		}
	}
	return nil
}
