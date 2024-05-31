package internal

import (
	"context"
	"fmt"
	"strconv"

	"github.com/VictorLowther/simplexml/dom"
	"github.com/VictorLowther/simplexml/search"
)

//go:generate stringer -type=bootConfigRole -linecomment -output=boot_string.go
type bootConfigRole int

const (
	isNext          bootConfigRole = 0
	isNextSingleUse bootConfigRole = 1
	isDefault       bootConfigRole = 2
	dmtfReserved    bootConfigRole = 3     // 3..32767
	vendorSpecified bootConfigRole = 32768 // 32768..65535
)

const resourceAMTBootSettingData = "http://intel.com/wbem/wscim/1/amt-schema/1/AMT_BootSettingData"

func (c *Client) setBootConfigRole(ctx context.Context, role bootConfigRole) error {
	bootConfigRef, err := c.getBootConfigSettingRef(ctx, "Intel(r) AMT: Boot Configuration 0")
	if err != nil {
		return err
	}

	message := c.WsmanClient.Invoke(resourceCIMBootService, "SetBootConfigRole")
	bootConfigSetting := message.MakeParameter("BootConfigSetting")
	bootConfigSetting.AddChildren(bootConfigRef.Children()...)
	message.AddParameter(bootConfigSetting)
	message.Parameters("Role", strconv.Itoa(int(role)))

	return sendMessageForReturnValueInt(ctx, message)
}

func (c *Client) changeBootOrder(ctx context.Context, items []string) error {
	message := c.WsmanClient.Invoke(resourceCIMBootConfigSetting, "ChangeBootOrder")

	if len(items) > 0 {
		// TODO: multiple?
		pxeEndpointRef, err := c.getBootSourceRef(ctx, "Intel(r) AMT: Force PXE Boot")
		if err != nil {
			return err
		}
		sourceParam := message.MakeParameter("Source")
		sourceParam.AddChildren(pxeEndpointRef.Children()...)
		message.AddParameter(sourceParam)
	}

	return sendMessageForReturnValueInt(ctx, message)
}

func (c *Client) getBootSettingData(ctx context.Context) ([]*dom.Element, error) {
	msg := c.WsmanClient.Get(resourceAMTBootSettingData)
	response, err := msg.Send(ctx)
	if err != nil {
		return nil, err
	}
	data := search.FirstTag("AMT_BootSettingData", resourceAMTBootSettingData, response.Body())
	if data == nil {
		return nil, fmt.Errorf("response was missing the AMT_BootSettingData")
	}
	return data.Children(), nil
}

func (c *Client) setBootSettingData(ctx context.Context) error {
	bootSettings, err := c.getBootSettingData(ctx)
	if err != nil {
		return err
	}

	settingsToKeep := []*dom.Element{}
	for _, setting := range bootSettings {
		switch setting.Name.Local {
		// omit these ones ... they are read-only parameters (per meshcommand implementation)
		case "WinREBootEnabled",
			"UEFILocalPBABootEnabled",
			"UEFIHTTPSBootEnabled",
			"SecureBootControlEnabled",
			"BootguardStatus",
			"OptionsCleared",
			"BIOSLastStatus",
			"UefiBootParametersArray":
			continue
		// gonna make sure these are set to "false"
		case "BIOSPause", "BIOSSetup":
			setting.Content = []byte("false")
			settingsToKeep = append(settingsToKeep, setting)
		// gonna make sure these are set to "0"
		case "BootMediaIndex":
			setting.Content = []byte("0")
			settingsToKeep = append(settingsToKeep, setting)
		default:
			settingsToKeep = append(settingsToKeep, setting)
		}
	}

	msg := c.WsmanClient.Put(resourceAMTBootSettingData)
	data := dom.Elem("AMT_BootSettingData", resourceAMTBootSettingData)
	data.AddChildren(settingsToKeep...)
	msg.SetBody(data)
	_, err = msg.Send(ctx)

	return err
}

func (c *Client) SetPXE(ctx context.Context) error {
	// clear existing boot order per meshcommander's implementation...
	// "Set the boot order to null, this is needed for some AMT versions that don't clear this automatically."
	// err := changeBootOrder(client, []string{})
	// if err != nil {
	// 	return err
	// }

	err := c.setBootSettingData(ctx)
	if err != nil {
		return err
	}

	err = c.setBootConfigRole(ctx, isNextSingleUse)
	if err != nil {
		return err
	}

	err = c.changeBootOrder(ctx, []string{"Intel(r) AMT: Force PXE Boot"})
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) getBootConfigSettingRef(ctx context.Context, name string) (*dom.Element, error) {
	return c.getEndpointReferenceByInstanceID(ctx, resourceCIMBootConfigSetting, name)
}

func (c *Client) getBootSourceRef(ctx context.Context, name string) (*dom.Element, error) {
	return c.getEndpointReferenceByInstanceID(ctx, resourceCIMBootSourceSetting, name)
}
