package diskencryption

// IgnitionConfig represents the ignitionconfig present in the config section of
// a machineconfig with disk encryption.
type IgnitionConfig struct {
	Storage struct {
		LUKS []struct {
			Clevis struct {
				Tpm  bool `json:"tpm2"`
				Tang []struct {
					Thumbprint string `json:"thumbprint"`
					URL        string `json:"url"`
				}
			} `json:"clevis"`
			Device     string   `json:"device"`
			Name       string   `json:"name"`
			Options    []string `json:"options"`
			WipeVolume bool     `json:"wipeVolume"`
		} `json:"luks"`
	} `json:"storage"`
}
