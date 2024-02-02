package internal

import (
	"context"
	"fmt"
	"strconv"

	"github.com/VictorLowther/simplexml/dom"
	"github.com/VictorLowther/simplexml/search"
	"github.com/jacobweinstock/iamt/internal/wsman"
)

const (
	resourceCIMAssociatedPowerManagementService = "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_AssociatedPowerManagementService"
	resourceCIMBootConfigSetting                = "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_BootConfigSetting"
	resourceCIMBootService                      = "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_BootService"
	resourceCIMBootSourceSetting                = "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_BootSourceSetting"
	resourceCIMPowerManagementService           = "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_PowerManagementService"
	resourceCIMComputerSystem                   = "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_ComputerSystem"
)

func (c *Client) getEndpointReferenceBySelector(ctx context.Context, namespace string, selectorName string, selectorValue string) (*dom.Element, error) {
	message := c.WsmanClient.EnumerateEPR(namespace)

	response, err := message.Send(ctx)
	if err != nil {
		return nil, err
	}
	items, err := response.EnumItems()
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		selector := search.First(search.Attr("Name", "*", selectorName), item.Descendants())
		if selector == nil || string(selector.Content) != selectorValue {
			continue
		}
		return item, nil
	}
	return nil, fmt.Errorf("could not find endpoint reference with selector %s=%s", selectorName, selectorValue)
}

func (c *Client) getEndpointReferenceByInstanceID(ctx context.Context, namespace string, instanceID string) (*dom.Element, error) {
	return c.getEndpointReferenceBySelector(ctx, namespace, "InstanceID", instanceID)
}

func (c *Client) getComputerSystemRef(ctx context.Context, name string) (*dom.Element, error) {
	return c.getEndpointReferenceBySelector(ctx, resourceCIMComputerSystem, "Name", name)
}

func getReturnValueInt(response *wsman.Message) (int, error) {
	returnElement := search.FirstTag("ReturnValue", "*", response.AllBodyElements())
	if returnElement == nil {
		return -1, fmt.Errorf("no ReturnValue found in the response")
	}
	return strconv.Atoi(string(returnElement.Content))
}

func sendMessageForReturnValueInt(ctx context.Context, message *wsman.Message) error {
	response, err := message.Send(ctx)
	if err != nil {
		return err
	}
	returnValue, err := getReturnValueInt(response)
	if err != nil {
		return err
	}

	if returnValue == 0 {
		return nil
	}

	return fmt.Errorf("received invalid return value %d", returnValue)
}
