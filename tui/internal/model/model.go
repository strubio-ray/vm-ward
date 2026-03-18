package model

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/strubio-ray/vm-ward/tui/internal/ui"
	"github.com/strubio-ray/vm-ward/tui/internal/vmw"
)

const refreshInterval = 30 * time.Second
const toastTTL = 3 * time.Second
const errorToastTTL = 10 * time.Second

// Messages
type tickMsg time.Time
type statusMsg struct {
	data vmw.StatusResponse
	err  error
}
type actionMsg struct {
	action string
	vmName string
	err    error
}
type sweepMsg struct {
	err error
}

// Model is the bubbletea Model for the vm-ward TUI.
type Model struct {
	client vmw.VMClient

	status vmw.StatusResponse
	vms    []vmw.VM
	err    error

	cursor      int
	selectedID  string
	lastRefresh time.Time
	now         time.Time
	refreshing  bool

	state         ViewState
	confirmAction string
	confirmVM     *vmw.VM

	pickerCursor int

	toast    *ui.Toast
	sweeping bool

	pendingAction string
	pendingVMName string
	pendingVMID   string

	width  int
	height int
}

// New creates a new Model.
func New(client vmw.VMClient) Model {
	return Model{
		client:       client,
		now:          time.Now(),
		pickerCursor: ui.DefaultPresetIndex,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchStatus(m.client),
		tickEverySecond(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.now = time.Time(msg)
		// Check if it's time for a full refresh
		var cmd tea.Cmd
		if !m.refreshing && time.Since(m.lastRefresh) >= refreshInterval {
			cmd = fetchStatus(m.client)
			m.refreshing = true
		}
		// Clear expired toast
		if m.toast != nil && !m.toast.IsVisible(m.now) {
			m.toast = nil
		}
		return m, tea.Batch(cmd, tickEverySecond())

	case statusMsg:
		m.refreshing = false
		m.lastRefresh = time.Now()
		m.now = time.Now()
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.err = nil
			m.status = msg.data
			m.vms = sortVMs(msg.data.VMs)
			m.preserveCursor()
		}
		return m, nil

	case actionMsg:
		m.pendingAction = ""
		m.pendingVMName = ""
		m.pendingVMID = ""
		if msg.err != nil {
			detail := errorDetail(msg.err, m.width-6, 8)
			m.toast = ui.NewToast(fmt.Sprintf("Error: %s %s\n%s", msg.action, msg.vmName, detail), true, errorToastTTL)
		} else {
			m.toast = ui.NewToast(fmt.Sprintf("%s %s", capitalizeAction(msg.action), msg.vmName), false, toastTTL)
		}
		// Auto-refresh after action
		m.refreshing = true
		return m, fetchStatus(m.client)

	case sweepMsg:
		m.sweeping = false
		if msg.err != nil {
			detail := errorDetail(msg.err, m.width-6, 8)
			m.toast = ui.NewToast(fmt.Sprintf("Sweep failed\n%s", detail), true, errorToastTTL)
		} else {
			m.toast = ui.NewToast("Sweep complete", false, toastTTL)
		}
		m.refreshing = true
		return m, fetchStatus(m.client)

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case StateConfirm:
		return m.handleConfirmKey(msg)
	case StateConfirmProvision:
		return m.handleProvisionKey(msg)
	case StatePicker:
		return m.handlePickerKey(msg)
	default:
		return m.handleNormalKey(msg)
	}
}

func (m Model) handleNormalKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case matchKey(msg, "q", "ctrl+c"):
		return m, tea.Quit

	case matchKey(msg, "up", "k"):
		if m.cursor > 0 {
			m.cursor--
			// Skip unmanaged VMs
			for m.cursor > 0 && !m.vms[m.cursor].Managed {
				m.cursor--
			}
			if !m.vms[m.cursor].Managed {
				m.cursor++
			}
			m.saveSelectedID()
		}
		return m, nil

	case matchKey(msg, "down", "j"):
		last := m.lastSelectableIndex()
		if m.cursor < last {
			m.cursor++
			// Skip unmanaged VMs
			for m.cursor <= last && !m.vms[m.cursor].Managed {
				m.cursor++
			}
			if m.cursor > last {
				m.cursor = last
			}
			m.saveSelectedID()
		}
		return m, nil

	case matchKey(msg, "r"):
		if !m.refreshing {
			m.refreshing = true
			return m, fetchStatus(m.client)
		}
		return m, nil

	case matchKey(msg, "h"):
		if vm := m.selectedVM(); vm != nil && vm.State == "running" && !m.hasPendingAction(vm.ID) {
			m.state = StateConfirm
			m.confirmAction = "halt"
			m.confirmVM = vm
		}
		return m, nil

	case matchKey(msg, "d"):
		if vm := m.selectedVM(); vm != nil && !m.hasPendingAction(vm.ID) {
			m.state = StateConfirm
			m.confirmAction = "destroy"
			m.confirmVM = vm
		}
		return m, nil

	case matchKey(msg, "x"):
		if vm := m.selectedVM(); vm != nil && !m.hasPendingAction(vm.ID) {
			m.state = StateConfirm
			m.confirmAction = "exempt"
			m.confirmVM = vm
		}
		return m, nil

	case matchKey(msg, "i"):
		if vm := m.selectedVM(); vm != nil && !m.hasPendingAction(vm.ID) {
			m.state = StateConfirm
			m.confirmAction = "set indefinite"
			m.confirmVM = vm
		}
		return m, nil

	case matchKey(msg, "e"):
		if vm := m.selectedVM(); vm != nil && !m.hasPendingAction(vm.ID) {
			m.state = StatePicker
			m.confirmVM = vm
			m.pickerCursor = ui.DefaultPresetIndex
		}
		return m, nil

	case matchKey(msg, "s"):
		if !m.sweeping {
			m.sweeping = true
			return m, doSweep(m.client)
		}
		return m, nil

	case matchKey(msg, "u"):
		if vm := m.selectedVM(); vm != nil && vm.TemplateVersion != nil && !m.hasPendingAction(vm.ID) {
			m.state = StateConfirm
			m.confirmAction = "update template for"
			m.confirmVM = vm
		}
		return m, nil

	case matchKey(msg, "shift+u"):
		if m.pendingAction == "" {
			m.state = StateConfirm
			m.confirmAction = "update all templates for"
			m.confirmVM = &vmw.VM{Name: "all VMs", Managed: true}
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case matchKey(msg, "y"):
		vm := m.confirmVM
		action := m.confirmAction
		// For update actions, ask about provisioning before dispatching
		if action == "update template for" || action == "update all templates for" {
			m.state = StateConfirmProvision
			return m, nil
		}
		m.pendingAction = action
		m.pendingVMName = vm.Name
		m.pendingVMID = vm.ID
		m.state = StateNormal
		m.confirmVM = nil
		return m, doAction(m.client, action, vm)

	case matchKey(msg, "n", "escape"):
		m.state = StateNormal
		m.confirmVM = nil
		return m, nil
	}
	return m, nil
}

func (m Model) handleProvisionKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case matchKey(msg, "y"):
		vm := m.confirmVM
		action := m.confirmAction + " --provision"
		m.pendingAction = action
		m.pendingVMName = vm.Name
		m.pendingVMID = vm.ID
		m.state = StateNormal
		m.confirmVM = nil
		return m, doAction(m.client, action, vm)

	case matchKey(msg, "n"):
		vm := m.confirmVM
		action := m.confirmAction
		m.pendingAction = action
		m.pendingVMName = vm.Name
		m.pendingVMID = vm.ID
		m.state = StateNormal
		m.confirmVM = nil
		return m, doAction(m.client, action, vm)

	case matchKey(msg, "escape"):
		m.state = StateNormal
		m.confirmVM = nil
		return m, nil
	}
	return m, nil
}

func (m Model) handlePickerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case matchKey(msg, "left", "h"):
		if m.pickerCursor > 0 {
			m.pickerCursor--
		}
		return m, nil

	case matchKey(msg, "right", "l"):
		if m.pickerCursor < len(ui.DurationPresets)-1 {
			m.pickerCursor++
		}
		return m, nil

	case matchKey(msg, "enter"):
		vm := m.confirmVM
		duration := ui.DurationPresets[m.pickerCursor]
		m.pendingAction = "extend"
		m.pendingVMName = vm.Name
		m.pendingVMID = vm.ID
		m.state = StateNormal
		m.confirmVM = nil
		return m, doExtend(m.client, vm, duration)

	case matchKey(msg, "escape"):
		m.state = StatePicker
		m.state = StateNormal
		m.confirmVM = nil
		return m, nil
	}
	return m, nil
}

func (m Model) View() tea.View {
	if m.width == 0 {
		v := tea.NewView("Loading...")
		v.AltScreen = true
		return v
	}

	var b strings.Builder

	// Error banner
	if m.err != nil {
		b.WriteString(ui.Red.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	// Daemon warning
	if m.status.Daemon.State != "running" && m.status.Daemon.State != "" {
		b.WriteString(ui.Yellow.Render(fmt.Sprintf("  Daemon: %s", m.status.Daemon.State)))
		b.WriteString("\n")
	}

	// Sweeping indicator
	if m.sweeping {
		b.WriteString(ui.Yellow.Render("  Sweeping..."))
		b.WriteString("\n")
	}

	// Action in-progress indicator
	if m.pendingAction != "" {
		label := fmt.Sprintf("  %s %s...", progressLabel(m.pendingAction), m.pendingVMName)
		b.WriteString(ui.Yellow.Render(label))
		b.WriteString("\n")
	}

	// Table header
	b.WriteString(ui.RenderHeader())
	b.WriteString("\n")
	b.WriteString(ui.RenderSeparator(m.width))
	b.WriteString("\n")

	if len(m.vms) == 0 {
		b.WriteString(ui.Dim.Render("  No Vagrant VMs found"))
		b.WriteString("\n")
	} else {
		// Active managed VMs
		anyRendered := false
		for i, vm := range m.vms {
			if !vm.Managed || vm.Section == "halted" {
				continue
			}
			b.WriteString(ui.RenderRow(vm, m.now, i == m.cursor))
			b.WriteString("\n")
			anyRendered = true
		}

		// Halted managed VMs
		hasHalted := false
		for _, vm := range m.vms {
			if vm.Managed && vm.Section == "halted" {
				hasHalted = true
				break
			}
		}
		if hasHalted {
			if anyRendered {
				b.WriteString("\n")
			}
			b.WriteString(ui.RenderSectionHeader("RECENTLY HALTED / EXPIRED"))
			b.WriteString("\n")
			b.WriteString(ui.RenderSeparator(m.width))
			b.WriteString("\n")
			for i, vm := range m.vms {
				if !vm.Managed || vm.Section != "halted" {
					continue
				}
				b.WriteString(ui.RenderRow(vm, m.now, i == m.cursor))
				b.WriteString("\n")
			}
			anyRendered = true
		}

		// Unmanaged VMs (non-selectable, dimmed)
		hasUnmanaged := false
		for _, vm := range m.vms {
			if !vm.Managed {
				hasUnmanaged = true
				break
			}
		}
		if hasUnmanaged {
			if anyRendered {
				b.WriteString("\n")
			}
			b.WriteString(ui.RenderSectionHeader("UNMANAGED VMs"))
			b.WriteString("\n")
			b.WriteString(ui.RenderSeparator(m.width))
			b.WriteString("\n")
			for _, vm := range m.vms {
				if vm.Managed {
					continue
				}
				b.WriteString(ui.RenderRow(vm, m.now, false))
				b.WriteString("\n")
			}
		}
	}

	// Toast
	if m.toast != nil && m.toast.IsVisible(m.now) {
		b.WriteString("\n")
		b.WriteString(m.toast.Render())
		b.WriteString("\n")
	}

	// Confirmation or picker bar
	switch m.state {
	case StateConfirm:
		b.WriteString("\n")
		b.WriteString(ui.RenderConfirm(m.confirmAction, m.confirmVM.Name))
		b.WriteString("\n")
	case StateConfirmProvision:
		b.WriteString("\n")
		b.WriteString(ui.RenderProvisionConfirm())
		b.WriteString("\n")
	case StatePicker:
		b.WriteString("\n")
		b.WriteString(ui.RenderPicker(m.pickerCursor, m.confirmVM.Name))
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(ui.RenderFooter(m.width, m.status.LastSweep, m.lastRefresh, m.now))

	// Pad to fill terminal height to prevent flickering
	rendered := b.String()
	lines := strings.Count(rendered, "\n") + 1
	if lines < m.height {
		rendered += strings.Repeat("\n", m.height-lines-1)
	}

	v := tea.NewView(lipgloss.NewStyle().MaxWidth(m.width).Render(rendered))
	v.AltScreen = true
	return v
}

// Helpers

func (m *Model) hasPendingAction(vmID string) bool {
	return m.pendingVMID != "" && m.pendingVMID == vmID
}

func (m *Model) selectedVM() *vmw.VM {
	if m.cursor >= 0 && m.cursor < len(m.vms) && m.vms[m.cursor].Managed {
		return &m.vms[m.cursor]
	}
	return nil
}

func (m *Model) saveSelectedID() {
	if vm := m.selectedVM(); vm != nil {
		m.selectedID = vm.ID
	}
}

func (m *Model) preserveCursor() {
	if m.selectedID == "" {
		m.clampCursor()
		return
	}
	for i, vm := range m.vms {
		if vm.ID == m.selectedID {
			m.cursor = i
			return
		}
	}
	m.clampCursor()
}

// sortVMs orders VMs so active section comes before halted, matching render order.
func sortVMs(vms []vmw.VM) []vmw.VM {
	sorted := make([]vmw.VM, len(vms))
	copy(sorted, vms)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sectionOrder(sorted[i]) < sectionOrder(sorted[j])
	})
	return sorted
}

func sectionOrder(vm vmw.VM) int {
	if !vm.Managed {
		return 2 // unmanaged always last
	}
	switch vm.Section {
	case "halted":
		return 1
	default:
		return 0
	}
}

// lastSelectableIndex returns the index of the last managed VM.
func (m *Model) lastSelectableIndex() int {
	for i := len(m.vms) - 1; i >= 0; i-- {
		if m.vms[i].Managed {
			return i
		}
	}
	return 0
}

func (m *Model) clampCursor() {
	last := m.lastSelectableIndex()
	if m.cursor > last {
		m.cursor = last
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// Commands

func fetchStatus(client vmw.VMClient) tea.Cmd {
	return func() tea.Msg {
		data, err := client.Status()
		return statusMsg{data, err}
	}
}

func tickEverySecond() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func doAction(client vmw.VMClient, action string, vm *vmw.VM) tea.Cmd {
	return func() tea.Msg {
		var err error
		switch action {
		case "halt":
			err = client.Halt(vm.ID)
		case "destroy":
			err = client.Destroy(vm.ID)
		case "exempt":
			err = client.Exempt(vm.ID)
		case "set indefinite":
			err = client.Extend(vm.ID, "indefinite")
		case "update template for":
			err = client.Update(vm.ID, false)
		case "update template for --provision":
			err = client.Update(vm.ID, true)
		case "update all templates for":
			err = client.UpdateAll(false)
		case "update all templates for --provision":
			err = client.UpdateAll(true)
		}
		return actionMsg{action, vm.Name, err}
	}
}

func doExtend(client vmw.VMClient, vm *vmw.VM, duration string) tea.Cmd {
	return func() tea.Msg {
		err := client.Extend(vm.ID, duration)
		return actionMsg{"Extended", vm.Name + " by " + duration, err}
	}
}

func doSweep(client vmw.VMClient) tea.Cmd {
	return func() tea.Msg {
		err := client.Sweep()
		return sweepMsg{err}
	}
}

// errorDetail formats a multi-line error for toast display.
// Each line is trimmed and truncated to maxWidth. Empty lines are skipped.
// At most maxLines lines are kept.
func errorDetail(err error, maxWidth, maxLines int) string {
	var lines []string
	for _, line := range strings.Split(err.Error(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if maxWidth > 3 && len(line) > maxWidth {
			line = line[:maxWidth-3] + "..."
		}
		lines = append(lines, line)
		if len(lines) >= maxLines {
			break
		}
	}
	return strings.Join(lines, "\n")
}

func progressLabel(s string) string {
	switch s {
	case "halt":
		return "Halting"
	case "destroy":
		return "Destroying"
	case "exempt":
		return "Exempting"
	case "set indefinite":
		return "Setting indefinite on"
	case "extend":
		return "Extending"
	case "update template for", "update template for --provision":
		return "Updating template for"
	case "update all templates for", "update all templates for --provision":
		return "Updating all templates for"
	default:
		return s
	}
}

func capitalizeAction(s string) string {
	switch s {
	case "halt":
		return "Halted"
	case "destroy":
		return "Destroyed"
	case "exempt":
		return "Exempted"
	case "set indefinite":
		return "Set indefinite on"
	case "update template for", "update template for --provision":
		return "Updated template for"
	case "update all templates for", "update all templates for --provision":
		return "Updated all templates for"
	default:
		return s
	}
}
