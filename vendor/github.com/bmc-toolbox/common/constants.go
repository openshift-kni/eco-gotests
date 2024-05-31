package common

import "strings"

const (
	VendorDell                  = "dell"
	VendorMicron                = "micron"
	VendorAsrockrack            = "asrockrack"
	VendorSupermicro            = "supermicro"
	VendorHPE                   = "hp"
	VendorQuanta                = "quanta"
	VendorGigabyte              = "gigabyte"
	VendorIntel                 = "intel"
	VendorLSI                   = "lsi"
	VendorHGST                  = "hgst"
	VendorPacket                = "packet"
	VendorMellanox              = "mellanox"
	VendorToshiba               = "toshiba"
	VendorAmericanMegatrends    = "american megatrends"
	VendorBroadcom              = "broadcom"
	VendorInfineon              = "infineon"
	VendorAMD                   = "amd"
	VendorHynix                 = "hynix"
	VendorSamsung               = "samsung"
	VendorMarvell               = "marvell"
	SystemManufacturerUndefined = "To Be Filled By O.E.M."

	// Generic component slugs
	// Slugs are set on Device types to identify the type of component
	SlugBackplaneExpander     = "Backplane-Expander"
	SlugChassis               = "Chassis"
	SlugTPM                   = "TPM"
	SlugGPU                   = "GPU"
	SlugCPU                   = "CPU"
	SlugPhysicalMem           = "PhysicalMemory"
	SlugStorageController     = "StorageController"
	SlugStorageControllers    = "StorageControllers"
	SlugBMC                   = "BMC"
	SlugBIOS                  = "BIOS"
	SlugDrive                 = "Drive"
	SlugDrives                = "Drives"
	SlugDriveTypePCIeNVMEeSSD = "NVMe-PCIe-SSD"
	SlugDriveTypeSATASSD      = "Sata-SSD"
	SlugDriveTypeSATAHDD      = "Sata-HDD"
	SlugNIC                   = "NIC"
	SlugNICs                  = "NICs"
	SlugPSU                   = "Power-Supply"
	SlugPSUs                  = "Power-Supplies"
	SlugCPLD                  = "CPLD"
	SlugEnclosure             = "Enclosure"
	SlugMainboard             = "Mainboard"
	SlugUnknown               = "unknown"

	// Smart status
	SmartStatusOK      = "ok"
	SmartStatusFailed  = "failed"
	SmartStatusUnknown = "unknown"

	VendorMarvellPciID = "1b4b"

	// RAID implementations
	SlugRAIDImplLinuxSoftware = "linuxsw"
	SlugRAIDImplZFS           = "zfs"
	SlugRAIDImplHardware      = "hardware"
)

// FormatVendorName compares the given strings to identify and returned a known
// vendor name. When a match is not found, the string is returned as is.
//
// Note: This method will most likely return incorrect matches if the given
// vendor string is too short and or not unique enough.
//
// nolint:gocyclo // This list is expected to be long.
func FormatVendorName(name string) string {
	v := strings.TrimSpace(strings.ToLower(name))

	switch v {
	case "hp", "hpe":
		return VendorHPE
	case "ami":
		return VendorAmericanMegatrends
	case "lsi":
		return VendorLSI
	case "amd":
		return VendorAMD
	}

	switch {
	case strings.Contains(v, VendorAsrockrack):
		return VendorAsrockrack
	case strings.Contains(v, VendorDell):
		return VendorDell
	case strings.Contains(v, VendorSupermicro):
		return VendorSupermicro
	case strings.Contains(v, VendorQuanta):
		return VendorQuanta
	case strings.Contains(v, VendorGigabyte):
		return VendorGigabyte
	case strings.Contains(v, VendorIntel):
		return VendorIntel
	case strings.Contains(v, VendorPacket):
		return VendorPacket
	case strings.Contains(v, VendorHynix):
		return VendorHynix
	case strings.Contains(v, VendorInfineon):
		return VendorInfineon
	case strings.Contains(v, VendorBroadcom):
		return VendorBroadcom
	case strings.Contains(v, VendorMellanox):
		return VendorMellanox
	case strings.Contains(v, VendorHGST):
		return VendorHGST
	case strings.Contains(v, VendorToshiba):
		return VendorToshiba
	case strings.Contains(v, VendorMicron):
		return VendorMicron
	case strings.Contains(v, VendorAmericanMegatrends):
		return VendorAmericanMegatrends
	case strings.Contains(v, VendorSamsung):
		return VendorSamsung
	case strings.Contains(v, VendorMarvell):
		return VendorMarvell
	default:
		return name
	}
}

// Return the product vendor name, given a product name/model string
func VendorFromString(s string) string {
	s = strings.ToLower(s)

	switch {
	case strings.Contains(s, "dell"):
		return VendorDell
	case strings.Contains(s, "lsi3008-it"):
		return VendorLSI
	case strings.Contains(s, "hgst "):
		return VendorHGST
	case strings.Contains(s, "intel "):
		return VendorIntel
	case strings.Contains(s, "micron_"), strings.HasPrefix(s, "mtfd"):
		return VendorMicron
	case strings.Contains(s, "toshiba"):
		return VendorToshiba
	case strings.Contains(s, "connectx4lx"):
		return VendorMellanox
	case strings.Contains(s, "infineon"):
		return VendorInfineon
	case strings.Contains(s, VendorMarvell), strings.Contains(s, VendorMarvellPciID):
		return VendorMarvell
	default:
		return ""
	}
}

// Return a normalized product name given a product name
//
// nolint:gocyclo // This list is expected to be long.
func FormatProductName(s string) string {
	switch s {
	case "PowerEdge R6515":
		return "r6515"
	case "PowerEdge R640":
		return "r640"
	case "PowerEdge R6415":
		return "r6415"
	case "PowerEdge R750":
		return "r750"
	case "PowerEdge C6320":
		return "c6320"
	case "SYS-5019C-MR-PH004", "SYS-5019C-MR":
		return "x11scm-f"
	case "SYS-5039MS-H12TRF":
		return "x11sse-f"
	case "SYS-510T-MR-EI018", "SYS-510T-MR1-EI018":
		return "x12sth-sys"
	case "SSG-6029P-E1CR12L-PH004":
		return "x11dph-t"
	case "SSG-110P-NTR10", "SSG-110P-NTR10-EI018":
		return "x12spo-ntf"
	default:
		return s
	}
}
