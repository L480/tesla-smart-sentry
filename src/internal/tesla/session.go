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

func Execute(ctx context.Context, c Config, vcsecOnly bool, cmd func(*vehicle.Vehicle) error) error {
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
		var domains []protocol.Domain
		if vcsecOnly {
			domains = append(domains, protocol.DomainVCSEC)
		} else {
			domains = append(domains, protocol.DomainVCSEC)
			domains = append(domains, protocol.DomainInfotainment)
		}
		if err := car.StartSession(ctx, domains); err != nil {
			return fmt.Errorf("failed to perform handshake with vehicle: %s", err)
		}
		if err := cmd(car); err != nil {
			return validateResult(err)
		}
	} else {
		if err := car.Wakeup(ctx); err != nil {
			return fmt.Errorf("failed to wake up vehicle: %s", err)
		}
		for {
			if err := car.StartSession(ctx, nil); err != nil {
				return fmt.Errorf("failed to perform handshake with vehicle: %s", err)
			}
			if err := car.Ping(ctx); err != nil {
				time.Sleep(1 * time.Second)
				continue
			}
			break
		}
		if err := cmd(car); err != nil {
			return validateResult(err)
		}
	}
	return nil
}
