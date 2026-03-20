package vmw

// StatusResponse is the top-level JSON returned by vmw status --json.
type StatusResponse struct {
	Daemon          DaemonInfo `json:"daemon"`
	LastSweep       *int64     `json:"last_sweep"`
	RecentEvents    []Event    `json:"recent_events"`
	VMs             []VM       `json:"vms"`
	CPUThreshold    *int       `json:"cpu_threshold"`
	ActivityEnabled *bool      `json:"activity_enabled"`
}

type DaemonInfo struct {
	State string `json:"state"`
	PID   *int   `json:"pid"`
}

type Event struct {
	Ts        int64  `json:"ts"`
	Type      string `json:"type"`
	VMName    string `json:"vm_name"`
	MachineID string `json:"machine_id"`
	Detail    string `json:"detail"`
}

type VM struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Path         string  `json:"path"`
	State        string  `json:"state"`
	Lease        string  `json:"lease"`
	Remaining    string  `json:"remaining"`
	ExpiresAt    *int64  `json:"expires_at"`
	Duration     *string `json:"duration"`
	LastActive   *int64  `json:"last_active"`
	HaltedAt     *int64  `json:"halted_at"`
	LastActivity *string `json:"last_activity"`
	CPUPercent   *int    `json:"cpu_percent"`
	Managed         bool    `json:"managed"`
	Section         string  `json:"section"`
	TemplateVersion *string `json:"template_version"`
}
