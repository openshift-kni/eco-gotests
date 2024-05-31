package internal

import (
	"context"
	"fmt"
	"strconv"

	"github.com/VictorLowther/simplexml/dom"
	"github.com/go-logr/logr"
	"github.com/jacobweinstock/iamt/internal/wsman"
)

//go:generate stringer -type=powerState -linecomment -output=power_string.go

type powerState int

// https://software.intel.com/sites/manageability/AMT_Implementation_and_Reference_Guide/HTMLDocuments/WS-Management_Class_Reference/CIM_AssociatedPowerManagementService.htm#powerState
// https://software.intel.com/sites/manageability/AMT_Implementation_and_Reference_Guide/default.htm?turl=WordDocuments%2Fgetsystempowerstate.htm
const (
	unknown                   powerState = 0
	other                     powerState = 1
	stateOn                   powerState = 2
	sleepLight                powerState = 3
	sleepDeep                 powerState = 4
	powerCycleOffSoft         powerState = 5
	offHard                   powerState = 6
	hibernateOffSoft          powerState = 7
	offSoft                   powerState = 8
	powerCycleOffHard         powerState = 9
	masterBusReset            powerState = 10
	diagnosticInterruptNMI    powerState = 11
	offSoftGraceful           powerState = 12
	offHardGraceful           powerState = 13
	masterBusResetGraceful    powerState = 14
	powerCycleOffSoftGraceful powerState = 15
	powerCycleOffHardGraceful powerState = 16
	diagnosticInterruptInit   powerState = 17
	// DMTF Reserved = ..
	// Vendor Specific = 0x7FFF..0xFFFF.
)

type powerStatus struct {
	AvailableRequestedpowerStates []powerState
	powerState                    powerState
	RequestedpowerState           powerState
}

func (c *Client) getPowerStatus(ctx context.Context) (*powerStatus, error) {
	// https://software.intel.com/sites/manageability/AMT_Implementation_and_Reference_Guide/default.htm?turl=WordDocuments%2Fgetsystempowerstate.htm
	message := c.WsmanClient.Enumerate(resourceCIMAssociatedPowerManagementService)

	response, err := message.Send(ctx)
	if err != nil {
		return nil, err
	}
	pmElms, err := getPowerManagementElements(response)
	if err != nil {
		return nil, err
	}

	status := &powerStatus{
		AvailableRequestedpowerStates: []powerState{},
	}
	for _, e := range pmElms {
		switch e.Name.Local {
		case "PowerState":
			val, err := strconv.Atoi(string(e.Content))
			if err != nil {
				return nil, err
			}
			status.powerState = powerState(val)
		case "RequestedPowerState":
			val, err := strconv.Atoi(string(e.Content))
			if err != nil {
				return nil, err
			}
			status.RequestedpowerState = powerState(val)
		case "AvailableRequestedPowerStates":
			val, err := strconv.Atoi(string(e.Content))
			if err != nil {
				return nil, err
			}
			status.AvailableRequestedpowerStates = append(status.AvailableRequestedpowerStates, powerState(val))
		}
	}

	return status, nil
}

func (c *Client) PowerOn(ctx context.Context) error {
	isOn, err := c.IsPoweredOn(ctx)
	if err != nil {
		return err
	}
	if isOn {
		return nil
	}

	return c.requestpowerState(ctx, stateOn)
}

func (c *Client) PowerOff(ctx context.Context) error {
	status, err := c.getPowerStatus(ctx)
	if err != nil {
		return err
	}
	if isPoweredOnGivenStatus(c.Log, status) {
		request := selectNextState(getPowerOffStates(), status.AvailableRequestedpowerStates)
		if request != unknown {
			return c.requestpowerState(ctx, request)
		}

		return fmt.Errorf("there is no implemented transition state to power off the machine from the current machine state %q. available states are: %v", status.powerState, status.AvailableRequestedpowerStates)
	}

	return nil
}

func (c *Client) PowerCycle(ctx context.Context) error {
	status, err := c.getPowerStatus(ctx)
	if err != nil {
		return err
	}

	if !isPoweredOnGivenStatus(c.Log, status) {
		return c.PowerOn(ctx)
	}

	request := selectNextState(getPowerCycleStates(), status.AvailableRequestedpowerStates)

	if request >= 0 {
		return c.requestpowerState(ctx, request)
	}

	return fmt.Errorf("there is no implemented transition state to power cycle the machine from the current machine state %d. available states are: %v", status.powerState, status.AvailableRequestedpowerStates)
}

func (c *Client) IsPoweredOn(ctx context.Context) (bool, error) {
	status, err := c.getPowerStatus(ctx)
	if err != nil {
		return false, err
	}
	return isPoweredOnGivenStatus(c.Log, status), nil
}

func isPoweredOnGivenStatus(log logr.Logger, status *powerStatus) bool {
	log.V(1).Info("states", "currentState", fmt.Sprintf("%v", status.powerState), "availableStates", fmt.Sprintf("%v", status.AvailableRequestedpowerStates))
	switch status.powerState {
	case stateOn:
		return true
	default:
		return false
	}
}

// https://software.intel.com/sites/manageability/AMT_Implementation_and_Reference_Guide/default.htm?turl=WordDocuments%2Fchangesystempowerstate.htm
func (c *Client) requestpowerState(ctx context.Context, requestedpowerState powerState) error {
	status, err := c.getPowerStatus(ctx)
	if err != nil {
		return err
	}
	if !containspowerState(status.AvailableRequestedpowerStates, requestedpowerState) {
		return fmt.Errorf("there is no implemented transition state to <%d> from the current machine state <%d>. available states are: %v", requestedpowerState, status.powerState, status.AvailableRequestedpowerStates)
	}
	c.Log.V(1).Info("sending request to machine", "PowerState", requestedpowerState)
	message := c.WsmanClient.Invoke(resourceCIMPowerManagementService, "RequestPowerStateChange")
	message.Parameters("PowerState", fmt.Sprint(int(requestedpowerState)))
	managedElement, err := c.makeManagedElement(ctx, message)
	if err != nil {
		return err
	}
	message.AddParameter(managedElement)

	response, err := message.Send(ctx)
	if err != nil {
		return err
	}

	body := response.GetBody(dom.Elem("RequestPowerStateChange_OUTPUT", resourceCIMPowerManagementService))
	if body == nil || len(body.Children()) != 1 {
		return fmt.Errorf("received unknown response requesting power state change: %v", response)
	}
	val, err := strconv.Atoi(string(body.Children()[0].Content))
	if err != nil {
		return err
	}
	c.Log.V(1).Info("RequestPowerState response", "response", val)

	return nil
}

func getPowerManagementElements(response *wsman.Message) ([]*dom.Element, error) {
	items, err := response.EnumItems()
	if err != nil {
		return nil, err
	}

	for _, e := range items {
		if e.Name.Local == "CIM_AssociatedPowerManagementService" && e.Name.Space == resourceCIMAssociatedPowerManagementService {
			return e.Children(), nil
		}
	}
	return nil, fmt.Errorf("did not receive %s enumeration item", "CIM_AssociatedPowerManagementService")
}

func (c *Client) makeManagedElement(ctx context.Context, message *wsman.Message) (*dom.Element, error) {
	managedSystemRef, err := c.getComputerSystemRef(ctx, "ManagedSystem")
	if err != nil {
		return nil, err
	}
	if managedSystemRef == nil {
		return nil, fmt.Errorf("could not retrieve the managed system endpoint reference")
	}
	managedElement := message.MakeParameter("ManagedElement")
	managedElement.AddChildren(managedSystemRef.Children()...)
	return managedElement, nil
}

func getPowerOffStates() []powerState {
	return []powerState{
		offSoftGraceful,
		offSoft,
		offHardGraceful,
		offHard,
	}
}

func getPowerCycleStates() []powerState {
	return []powerState{
		powerCycleOffSoftGraceful,
		powerCycleOffSoft,
		masterBusResetGraceful,
		powerCycleOffHardGraceful,
		powerCycleOffHard,
		masterBusReset,
	}
}

func selectNextState(requestedStates []powerState, availableStates []powerState) powerState {
	for _, a := range requestedStates {
		if containspowerState(availableStates, a) {
			return a
		}
	}
	return unknown
}

func containspowerState(s []powerState, e powerState) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
