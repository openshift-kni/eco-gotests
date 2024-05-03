package redfish

import (
	"errors"
	"strings"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranparam"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

// Connect uses the BMC information in RANConfig to return an APIClient for redfish.
func Connect() (*gofish.APIClient, error) {
	username := raninittools.RANConfig.BmcUsername
	password := raninittools.RANConfig.BmcPassword
	hosts := strings.Split(raninittools.RANConfig.BmcHosts, ",")

	if len(hosts) > 1 {
		glog.V(ranparam.LogLevel).Infof("Found multiple BmcHosts, using %s", hosts[0])
	}

	host := hosts[0]
	if !strings.HasPrefix(host, "https://") {
		host = "https://" + host
	}

	clientConfig := gofish.ClientConfig{
		Endpoint:  host,
		Username:  username,
		Password:  password,
		Insecure:  true,
		BasicAuth: true,
	}

	return gofish.Connect(clientConfig)
}

// GetVendor returns the vendor represented by the redfish service.
func GetVendor(client *gofish.APIClient) (string, error) {
	if client == nil {
		return "", errors.New("nil api client was provided")
	}

	service := client.GetService()
	if service == nil {
		return "", errors.New("api client returned nil service")
	}

	return service.Vendor, nil
}

// GetPowerUsage gets the power usage using the provided client.
func GetPowerUsage(client *gofish.APIClient) (float64, error) {
	if client == nil {
		return 0.0, errors.New("nil api client was provided")
	}

	service := client.GetService()
	if service == nil {
		return 0.0, errors.New("api client returned nil service")
	}

	chassisList, err := service.Chassis()
	if err != nil {
		return 0.0, err
	}

	// return from first chassis that has a power usage
	for _, chassis := range chassisList {
		if chassis == nil {
			continue
		}

		power, err := chassis.Power()
		if err != nil || power == nil || len(power.PowerControl) == 0 {
			continue
		}

		return float64(power.PowerControl[0].PowerConsumedWatts), nil
	}

	return 0.0, errors.New("could not find chassis with power reading")
}

// PowerOff performs a graceful shutdown using the provided client.
func PowerOff(client *gofish.APIClient) error {
	if client == nil {
		return errors.New("nil api client was provided")
	}

	service := client.GetService()
	if service == nil {
		return errors.New("api client returned nil service")
	}

	systems, err := service.Systems()
	if err != nil {
		return err
	}

	// perform graceful shutdown on first available system
	for _, system := range systems {
		if system == nil {
			continue
		}

		return system.Reset(redfish.GracefulShutdownResetType)
	}

	return errors.New("could not find system to perform graceful shutdown")
}

// PowerOn powers on the machine using the provided client.
func PowerOn(client *gofish.APIClient) error {
	if client == nil {
		return errors.New("nil api client was provided")
	}

	service := client.GetService()
	if service == nil {
		return errors.New("api client returned nil service")
	}

	systems, err := service.Systems()
	if err != nil {
		return err
	}

	// power on on first available system
	for _, system := range systems {
		if system == nil {
			continue
		}

		return system.Reset(redfish.OnResetType)
	}

	return errors.New("could not find system to power on")
}
