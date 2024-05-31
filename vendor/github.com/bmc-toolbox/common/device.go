// Package common is intended to provide a common data structure to model
// server hardware and its component attributes between libraries/tools.
package common

// Common holds attributes shared by all components
type Common struct {
	Oem          bool              `json:"oem"`
	Description  string            `json:"description,omitempty"`
	Vendor       string            `json:"vendor,omitempty"`
	Model        string            `json:"model,omitempty"`
	Serial       string            `json:"serial,omitempty"`
	ProductName  string            `json:"product_name,omitempty"`
	LogicalName  string            `json:"logical_name,omitempty"`
	PCIVendorID  string            `json:"pci_vendor_id,omitempty"`
	PCIProductID string            `json:"pci_product_id,omitempty"`
	Capabilities []*Capability     `json:"capabilities,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Firmware     *Firmware         `json:"firmware,omitempty"`
	Status       *Status           `json:"status,omitempty"`
}

// Device type is composed of various components
type Device struct {
	Common

	HardwareType       string               `json:"hardware_type,omitempty"`
	Chassis            string               `json:"chassis,omitempty"`
	BIOS               *BIOS                `json:"bios,omitempty"`
	BMC                *BMC                 `json:"bmc,omitempty"`
	Mainboard          *Mainboard           `json:"mainboard,omitempty"`
	CPLDs              []*CPLD              `json:"cplds"`
	TPMs               []*TPM               `json:"tpms,omitempty"`
	GPUs               []*GPU               `json:"gpus,omitempty"`
	CPUs               []*CPU               `json:"cpus,omitempty"`
	Memory             []*Memory            `json:"memory,omitempty"`
	NICs               []*NIC               `json:"nics,omitempty"`
	Drives             []*Drive             `json:"drives,omitempty"`
	StorageControllers []*StorageController `json:"storage_controller,omitempty"`
	PSUs               []*PSU               `json:"power_supplies,omitempty"`
	Enclosures         []*Enclosure         `json:"enclosures,omitempty"`
}

// NewDevice returns a pointer to an initialized Device type
func NewDevice() Device {
	return Device{
		BMC:                &BMC{NIC: &NIC{}},
		BIOS:               &BIOS{},
		Mainboard:          &Mainboard{},
		TPMs:               []*TPM{},
		CPLDs:              []*CPLD{},
		PSUs:               []*PSU{},
		NICs:               []*NIC{},
		GPUs:               []*GPU{},
		CPUs:               []*CPU{},
		Memory:             []*Memory{},
		Drives:             []*Drive{},
		StorageControllers: []*StorageController{},
		Enclosures:         []*Enclosure{},
	}
}

// Firmware struct holds firmware attributes of a device component
type Firmware struct {
	Installed  string            `json:"installed,omitempty"`
	Available  string            `json:"available,omitempty"`
	SoftwareID string            `json:"software_id,omitempty"`
	Previous   []*Firmware       `json:"previous,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// NewFirmwareObj returns a *Firmware object
func NewFirmwareObj() *Firmware {
	return &Firmware{Metadata: make(map[string]string)}
}

// Capability is a struct to describe a device/component capability and its status.
type Capability struct {
	Name        string `json:"Name"`
	Description string `json:"Description"`
	Enabled     bool   `json:"Enabled"`
}

// Status is the health status of a component
type Status struct {
	Health         string
	State          string
	PostCode       int    `json:"post_code,omitempty"`
	PostCodeStatus string `json:"post_code_status,omitempty"`
}

// GPU component
type GPU struct {
	Common
}

// Enclosure component
type Enclosure struct {
	Common

	ID          string    `json:"id,omitempty"`
	ChassisType string    `json:"chassis_type,omitempty"`
	Firmware    *Firmware `json:"firmware,omitempty"`
}

// TPM component
type TPM struct {
	Common

	InterfaceType string `json:"interface_type,omitempty"`
}

// CPLD component
type CPLD struct {
	Common
}

// PSU component
type PSU struct {
	Common

	ID                 string `json:"id,omitempty"`
	PowerCapacityWatts int64  `json:"power_capacity_watts,omitempty"`
}

// BIOS component
type BIOS struct {
	Common

	SizeBytes     int64 `json:"size_bytes,omitempty"`
	CapacityBytes int64 `json:"capacity_bytes,omitempty" diff:"immutable"`
}

// BMC component
type BMC struct {
	Common

	ID  string `json:"id,omitempty"`
	NIC *NIC   `json:"nic,omitempty"`
}

// CPU component
type CPU struct {
	Common

	ID           string `json:"id,omitempty"`
	Slot         string `json:"slot,omitempty"`
	Architecture string `json:"architecture,omitempty"`
	ClockSpeedHz int64  `json:"clock_speeed_hz,omitempty"`
	Cores        int    `json:"cores,omitempty"`
	Threads      int    `json:"threads,omitempty"`
}

// Memory component
type Memory struct {
	Common

	ID           string `json:"id,omitempty"`
	Slot         string `json:"slot,omitempty"`
	Type         string `json:"type,omitempty"`
	SizeBytes    int64  `json:"size_bytes,omitempty"`
	FormFactor   string `json:"form_factor,omitempty"`
	PartNumber   string `json:"part_number,omitempty"`
	ClockSpeedHz int64  `json:"clock_speed_hz,omitempty"`
}

// NIC component
type NIC struct {
	Common

	ID       string     `json:"id,omitempty"`
	NICPorts []*NICPort `json:"nic_ports,omitempty"`
}

// NICPort component
type NICPort struct {
	Common

	ID                   string `json:"id"`
	SpeedBits            int64  `json:"speed_bits,omitempty"`
	PhysicalID           string `json:"physid,omitempty"`
	BusInfo              string `json:"bus_info,omitempty"`
	ActiveLinkTechnology string `json:"active_link_technology,omitempty"`
	MacAddress           string `json:"macaddress,omitempty"`
	LinkStatus           string `json:"link_status,omitempty"`
	AutoNeg              bool   `json:"auto_neg,omitempty"`
	MTUSize              int    `json:"mtu_size,omitempty"`
}

// StorageController component
type StorageController struct {
	Common

	ID                           string `json:"id,omitempty"`
	SupportedControllerProtocols string `json:"supported_controller_protocol,omitempty"` // PCIe
	SupportedDeviceProtocols     string `json:"supported_device_protocol,omitempty"`     // Attached device protocols - SAS, SATA
	SupportedRAIDTypes           string `json:"supported_raid_types,omitempty"`
	PhysicalID                   string `json:"physid,omitempty"`
	BusInfo                      string `json:"bus_info,omitempty"`
	SpeedGbps                    int64  `json:"speed_gbps,omitempty"`
	MaxPhysicalDisks             int    `json:"max_physical_disks,omitempty"`
	MaxVirtualDisks              int    `json:"max_virtual_disks,omitempty"`
}

// Mainboard component
type Mainboard struct {
	Common

	PhysicalID string `json:"physid,omitempty"`
}

// Drive component
type Drive struct {
	Common

	ID                       string                  `json:"id,omitempty"`
	OemID                    string                  `json:"oem_id,omitempty"`
	Type                     string                  `json:"drive_type,omitempty"`
	StorageController        string                  `json:"storage_controller,omitempty"`
	BusInfo                  string                  `json:"bus_info,omitempty"`
	WWN                      string                  `json:"wwn,omitempty"`
	Protocol                 string                  `json:"protocol,omitempty"`
	SmartStatus              string                  `json:"smart_status,omitempty"`
	SmartErrors              []string                `json:"smart_errors,omitempty"`
	SmartAttributes          []*DriveSmartAttributes `json:"smart_attributes,omitempty"`
	CapacityBytes            int64                   `json:"capacity_bytes,omitempty"`
	BlockSizeBytes           int64                   `json:"block_size_bytes,omitempty"`
	CapableSpeedGbps         int64                   `json:"capable_speed_gbps,omitempty"`
	NegotiatedSpeedGbps      int64                   `json:"negotiated_speed_gbps,omitempty"`
	StorageControllerDriveID int                     `json:"storage_controller_drive_id,omitempty"`
}

// VirtualDisk models RAID arrays
type VirtualDisk struct {
	ID             string   `json:"id,omitempty"`
	Name           string   `json:"name,omitempty"`
	RaidType       string   `json:"raid_type,omitempty"`
	SizeBytes      int64    `json:"size_bytes,omitempty"`
	Status         string   `json:"status,omitempty"`
	PhysicalDrives []*Drive `json:"physical_drives,omitempty"`
}

// DriveSmartAttributes holds SMART attributes for a drive.
//
// comments on fields taken from https://www.smartmontools.org/browser/trunk/smartmontools/smartctl.8.in
type DriveSmartAttributes struct {
	// Name is the SMART attribute name.
	Name string `json:"name,omitempty"`

	// Each vendor uses their own algorithm to convert this "Raw" value to a "Normalized" value
	// in the range from 1 to 254. Please keep in mind that smartctl only reports the different Attribute types,
	// values, and thresholds as read from the device. It does not carry out the conversion between "Raw" and
	// "Normalized" values: this is done by the disk's firmware.
	NormalizedValue int `json:"normalized_value,omitempty"`

	// Each Attribute also has a "Worst" value shown under the heading "WORST".
	// This is the smallest (closest to failure) value that the disk has recorded at any time during its lifetime when SMART was enabled.
	// [Note however that some vendors firmware may actually increase the "Worst" value for some "rate-type" Attributes.]
	Worst int `json:"worst,omitempty"`

	// Each Attribute also has a Threshold value (whose range is 0 to 255) which is printed under the heading "THRESH".
	// If the Normalized value is less than or equal to the Threshold value, then the Attribute is said to have failed.
	// If the Attribute is a pre-failure Attribute, then disk failure is imminent.
	Threshold int `json:"threshold,omitempty"`

	// If the Normalized value is less than or equal to the Threshold value, then the Attribute is said to have failed.
	// If the Attribute is a pre-failure Attribute, then disk failure is imminent.
	PreFailure bool `json:"prefailure,omitempty"`

	// Some SMART attribute values are updated only during off-line data collection activities;
	// the rest are updated during normal operation of the device or during both normal operation and off-line testing.
	UpdatedOnline bool `json:"updated_online,omitempty"`
}
