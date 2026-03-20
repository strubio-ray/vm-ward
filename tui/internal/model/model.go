package model

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/strubio-ray/vm-ward/tui/internal/ui"
	"github.com/strubio-ray/vm-ward/tui/internal/vmw"
)

const refreshInterval = 30 * time.Second
const toastTTL = 3 * time.Second
const errorToastTTL = 10 * time.Second
const peekTimeout = 30 * time.Second

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

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
type peekMsg struct {
	vmName string
	raw    string
	err    error
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

	pickerCursor    int

	toast    *ui.Toast
	sweeping bool

	pendingAction string
	pendingVMName string
	pendingVMID   string

	peekVM        *vmw.VM
	peekRaw       string
	peekScroll    int
	peekLoading   bool
	peekCancel    context.CancelFunc
	peekStartTime time.Time

	keys keyMap
	help help.Model

	width  int
	height int
}

// New creates a new Model.
func New(client vmw.VMClient) Model {
	return Model{
		client:       client,
		now:          time.Now(),
		pickerCursor: ui.DefaultPresetIndex,
		keys:         newKeyMap(),
		help:         help.New(),
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
		m.help.SetWidth(msg.Width)
		m.clampCursor()
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

	case peekMsg:
		m.peekLoading = false
		m.peekCancel = nil
		if msg.err != nil {
			if errors.Is(msg.err, context.Canceled) || errors.Is(msg.err, context.DeadlineExceeded) {
				// User cancelled or timeout — only show toast if not already shown
				if m.peekVM != nil {
					if errors.Is(msg.err, context.DeadlineExceeded) {
						m.toast = ui.NewToast("Peek timed out", true, errorToastTTL)
					}
					m.peekVM = nil
				}
			} else {
				m.toast = ui.NewToast(fmt.Sprintf("Peek failed: %s", errorDetail(msg.err, m.width-6, 3)), true, errorToastTTL)
				m.peekVM = nil
			}
		} else {
			m.state = StatePeek
			m.peekRaw = msg.raw
			m.peekScroll = 0
		}
		return m, nil

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
	case StatePeek:
		return m.handlePeekKey(msg)
	case StateHelp:
		return m.handleHelpKey(msg)
	default:
		return m.handleNormalKey(msg)
	}
}

func (m Model) handleNormalKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.state = StateHelp
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
			// Skip non-selectable VMs (unmanaged or collapsed halted)
			for m.cursor > 0 && !m.isSelectable(m.cursor) {
				m.cursor--
			}
			if !m.isSelectable(m.cursor) {
				m.cursor++
			}
			m.saveSelectedID()
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		last := m.lastSelectableIndex()
		if m.cursor < last {
			m.cursor++
			// Skip non-selectable VMs (unmanaged or collapsed halted)
			for m.cursor <= last && !m.isSelectable(m.cursor) {
				m.cursor++
			}
			if m.cursor > last {
				m.cursor = last
			}
			m.saveSelectedID()
		}
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		if !m.refreshing {
			m.refreshing = true
			return m, fetchStatus(m.client)
		}
		return m, nil

	case key.Matches(msg, m.keys.Halt):
		if vm := m.selectedVM(); vm != nil && vm.State == "running" && !m.hasPendingAction(vm.ID) {
			m.state = StateConfirm
			m.confirmAction = "halt"
			m.confirmVM = vm
		}
		return m, nil

	case key.Matches(msg, m.keys.Destroy):
		if vm := m.selectedVM(); vm != nil && !m.hasPendingAction(vm.ID) {
			m.state = StateConfirm
			m.confirmAction = "destroy"
			m.confirmVM = vm
		}
		return m, nil

	case key.Matches(msg, m.keys.Exempt):
		if vm := m.selectedVM(); vm != nil && !m.hasPendingAction(vm.ID) {
			m.state = StateConfirm
			m.confirmAction = "exempt"
			m.confirmVM = vm
		}
		return m, nil

	case key.Matches(msg, m.keys.Indefinite):
		if vm := m.selectedVM(); vm != nil && !m.hasPendingAction(vm.ID) {
			m.state = StateConfirm
			m.confirmAction = "set indefinite"
			m.confirmVM = vm
		}
		return m, nil

	case key.Matches(msg, m.keys.Extend):
		if vm := m.selectedVM(); vm != nil && !m.hasPendingAction(vm.ID) {
			m.state = StatePicker
			m.confirmVM = vm
			m.pickerCursor = ui.DefaultPresetIndex
		}
		return m, nil

	case key.Matches(msg, m.keys.Sweep):
		if !m.sweeping {
			m.sweeping = true
			return m, doSweep(m.client)
		}
		return m, nil

	case key.Matches(msg, m.keys.Update):
		if vm := m.selectedVM(); vm != nil && vm.TemplateVersion != nil && !m.hasPendingAction(vm.ID) {
			m.state = StateConfirm
			m.confirmAction = "update template for"
			m.confirmVM = vm
		}
		return m, nil

	case key.Matches(msg, m.keys.UpdateAll):
		if m.pendingAction == "" {
			m.state = StateConfirm
			m.confirmAction = "update all templates for"
			m.confirmVM = &vmw.VM{Name: "all VMs", Managed: true}
		}
		return m, nil

	case key.Matches(msg, m.keys.Peek):
		if vm := m.selectedVM(); vm != nil && vm.State == "running" && !m.hasPendingAction(vm.ID) {
			if m.peekCancel != nil {
				m.peekCancel()
			}
			ctx, cancel := context.WithTimeout(context.Background(), peekTimeout)
			m.peekVM = vm
			m.peekLoading = true
			m.peekCancel = cancel
			m.peekStartTime = m.now
			return m, doPeek(ctx, m.client, vm)
		}
		return m, nil

	case key.Matches(msg, m.keys.Escape):
		if m.peekLoading && m.peekCancel != nil {
			m.peekCancel()
			m.peekLoading = false
			m.peekVM = nil
			m.peekCancel = nil
			m.toast = ui.NewToast("Peek cancelled", false, toastTTL)
			return m, nil
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.ConfirmYes):
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

	case key.Matches(msg, m.keys.ConfirmNo), key.Matches(msg, m.keys.Escape):
		m.state = StateNormal
		m.confirmVM = nil
		return m, nil
	}
	return m, nil
}

func (m Model) handleProvisionKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.ConfirmYes):
		vm := m.confirmVM
		action := m.confirmAction + " --provision"
		m.pendingAction = action
		m.pendingVMName = vm.Name
		m.pendingVMID = vm.ID
		m.state = StateNormal
		m.confirmVM = nil
		return m, doAction(m.client, action, vm)

	case key.Matches(msg, m.keys.ConfirmNo):
		vm := m.confirmVM
		action := m.confirmAction
		m.pendingAction = action
		m.pendingVMName = vm.Name
		m.pendingVMID = vm.ID
		m.state = StateNormal
		m.confirmVM = nil
		return m, doAction(m.client, action, vm)

	case key.Matches(msg, m.keys.Escape):
		m.state = StateNormal
		m.confirmVM = nil
		return m, nil
	}
	return m, nil
}

// handlePickerNav handles shared left/right/quit/esc navigation for horizontal pickers.
// Returns (model, cmd, handled). If handled is false, the caller should process the key.
func (m Model) handlePickerNav(cursor *int, maxIndex int, msg tea.KeyPressMsg) (Model, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit, true
	case key.Matches(msg, m.keys.PickerLeft):
		if *cursor > 0 {
			(*cursor)--
		}
		return m, nil, true
	case key.Matches(msg, m.keys.PickerRight):
		if *cursor < maxIndex {
			(*cursor)++
		}
		return m, nil, true
	case key.Matches(msg, m.keys.Escape):
		m.state = StateNormal
		m.confirmVM = nil
		return m, nil, true
	}
	return m, nil, false
}

func (m Model) handlePickerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m2, cmd, handled := m.handlePickerNav(&m.pickerCursor, len(ui.DurationPresets)-1, msg); handled {
		return m2, cmd
	}
	if key.Matches(msg, m.keys.PickerEnter) {
		vm := m.confirmVM
		duration := ui.DurationPresets[m.pickerCursor]
		m.pendingAction = "extend"
		m.pendingVMName = vm.Name
		m.pendingVMID = vm.ID
		m.state = StateNormal
		m.confirmVM = nil
		return m, doExtend(m.client, vm, duration)
	}
	return m, nil
}

func (m Model) handlePeekKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "ctrl+c":
		return m, tea.Quit

	case key.Matches(msg, m.keys.Escape), msg.String() == "q":
		if m.peekCancel != nil {
			m.peekCancel()
			m.peekCancel = nil
		}
		m.state = StateNormal
		m.peekVM = nil
		m.peekRaw = ""
		m.peekScroll = 0
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.peekScroll > 0 {
			m.peekScroll--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		m.peekScroll++
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		if m.peekVM != nil && !m.peekLoading {
			if m.peekCancel != nil {
				m.peekCancel()
			}
			ctx, cancel := context.WithTimeout(context.Background(), peekTimeout)
			m.peekLoading = true
			m.peekCancel = cancel
			m.peekStartTime = m.now
			return m, doPeek(ctx, m.client, m.peekVM)
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleHelpKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "ctrl+c":
		return m, tea.Quit
	case key.Matches(msg, m.keys.Help), key.Matches(msg, m.keys.Escape):
		m.state = StateNormal
		return m, nil
	case msg.String() == "q":
		m.state = StateNormal
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

	// Peek overlay (full-screen replacement)
	if m.state == StatePeek && m.peekRaw != "" {
		termLog, processes := ui.ParsePeekOutput(m.peekRaw)
		termRendered := ui.RenderTerminalLog(termLog, m.width-4, 24)
		peekElapsed := int(m.now.Sub(m.peekStartTime).Seconds())
		content := ui.RenderPeekOverlay(m.peekVM.Name, termRendered, processes,
			m.peekScroll, m.width, m.height, m.peekLoading, peekElapsed)
		lines := strings.Count(content, "\n") + 1
		if lines < m.height {
			content += strings.Repeat("\n", m.height-lines-1)
		}
		v := tea.NewView(lipgloss.NewStyle().MaxWidth(m.width).Render(content))
		v.AltScreen = true
		return v
	}

	// Help overlay (full-screen replacement)
	if m.state == StateHelp {
		content := ui.RenderHelpOverlay(m.help, m.keys, m.width, m.height)
		lines := strings.Count(content, "\n") + 1
		if lines < m.height {
			content += strings.Repeat("\n", m.height-lines-1)
		}
		v := tea.NewView(lipgloss.NewStyle().MaxWidth(m.width).Render(content))
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

	// Peek loading indicator
	if m.peekLoading && m.peekVM != nil && m.state != StatePeek {
		elapsed := int(m.now.Sub(m.peekStartTime).Seconds())
		frame := spinnerFrames[elapsed%len(spinnerFrames)]
		label := fmt.Sprintf("  %s Peeking %s... (%ds) — Esc to cancel", frame, m.peekVM.Name, elapsed)
		b.WriteString(ui.Yellow.Render(label))
		b.WriteString("\n")
	}

	// Table header
	b.WriteString(ui.RenderHeader(m.width))
	b.WriteString("\n")
	b.WriteString(ui.RenderSeparator(m.width))
	b.WriteString("\n")

	if len(m.vms) == 0 {
		b.WriteString(ui.Dim.Render("  No Vagrant VMs found"))
		b.WriteString("\n")
	} else {
		layout := m.computeLayout()

		// Active managed VMs
		anyRendered := false
		for i, vm := range m.vms {
			if !vm.Managed || vm.Section == "halted" {
				continue
			}
			b.WriteString(ui.RenderRow(vm, m.now, i == m.cursor, m.width))
			b.WriteString("\n")
			anyRendered = true
		}

		// Halted managed VMs
		switch layout.showHalted {
		case haltedFull:
			if layout.haltedCount > 0 {
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
					b.WriteString(ui.RenderRow(vm, m.now, i == m.cursor, m.width))
					b.WriteString("\n")
				}
				anyRendered = true
			}
		case haltedSummary:
			if anyRendered {
				b.WriteString("\n")
			}
			b.WriteString(ui.RenderCollapsedSummary(layout.haltedCount, "recently halted"))
			b.WriteString("\n")
			anyRendered = true
		case haltedNone:
			// skip
		}

		// Unmanaged VMs (non-selectable, dimmed)
		if layout.showUnmanaged && layout.unmanagedCount > 0 {
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
				b.WriteString(ui.RenderRow(vm, m.now, false, m.width))
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
	vm := m.selectedVM()
	m.keys.updateKeyStates(vm, m.pendingAction != "", m.sweeping)
	helpView := m.help.View(m.keys)
	b.WriteString("\n")
	b.WriteString(ui.RenderFooter(m.width, helpView, m.status.LastSweep, m.lastRefresh, m.now))

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

// haltedDisplay controls how the halted section renders.
type haltedDisplay string

const (
	haltedFull    haltedDisplay = "full"
	haltedSummary haltedDisplay = "summary"
	haltedNone    haltedDisplay = "none"
)

type sectionLayout struct {
	showHalted     haltedDisplay
	showUnmanaged  bool
	activeCount    int
	haltedCount    int
	unmanagedCount int
}

// computeLayout determines which sections to show based on terminal height.
func (m Model) computeLayout() sectionLayout {
	var active, halted, unmanaged int
	for _, vm := range m.vms {
		if !vm.Managed {
			unmanaged++
		} else if vm.Section == "halted" {
			halted++
		} else {
			active++
		}
	}

	layout := sectionLayout{
		activeCount:    active,
		haltedCount:    halted,
		unmanagedCount: unmanaged,
		showHalted:     haltedFull,
		showUnmanaged:  true,
	}

	if m.height == 0 || len(m.vms) == 0 {
		return layout
	}

	// Calculate fixed overhead (lines consumed by non-VM content).
	overhead := 0

	if m.err != nil {
		overhead += 2 // error line + blank
	}
	if m.status.Daemon.State != "running" && m.status.Daemon.State != "" {
		overhead++ // daemon warning
	}
	if m.sweeping {
		overhead++ // sweeping indicator
	}
	if m.pendingAction != "" {
		overhead++ // action progress
	}
	if m.peekLoading && m.peekVM != nil && m.state != StatePeek {
		overhead++ // peek loading
	}

	overhead += 2 // table header + separator

	if m.toast != nil && m.toast.IsVisible(m.now) {
		overhead += 1 + m.toast.Lines() // blank + toast lines
	}

	switch m.state {
	case StateConfirm, StateConfirmProvision, StatePicker:
		overhead += 2 // blank + confirm/picker line
	}

	overhead += 2 // blank + footer

	available := m.height - overhead - active

	// Cost of full halted section: header + separator + rows.
	// Plus a blank line if active VMs were rendered before it.
	haltedFullCost := 0
	if halted > 0 {
		haltedFullCost = 2 + halted // header + separator + rows
		if active > 0 {
			haltedFullCost++ // blank separator
		}
	}

	// Cost of halted summary: just 1 line (or 2 with blank separator).
	haltedSummaryCost := 0
	if halted > 0 {
		haltedSummaryCost = 1
		if active > 0 {
			haltedSummaryCost++ // blank separator
		}
	}

	// Cost of full unmanaged section.
	unmanagedFullCost := 0
	if unmanaged > 0 {
		unmanagedFullCost = 2 + unmanaged // header + separator + rows
		if active > 0 || halted > 0 {
			unmanagedFullCost++ // blank separator
		}
	}

	// Decision logic: try to fit everything, then progressively collapse.
	if available >= haltedFullCost+unmanagedFullCost {
		// Everything fits.
		return layout
	}

	// Hide unmanaged first.
	layout.showUnmanaged = false

	if available >= haltedFullCost {
		return layout
	}

	// Collapse halted to summary.
	if halted > 0 && available >= haltedSummaryCost {
		layout.showHalted = haltedSummary
		return layout
	}

	// Hide halted entirely.
	if halted > 0 {
		layout.showHalted = haltedNone
	}

	return layout
}

// isSelectable returns whether the VM at index idx can be selected by the cursor.
func (m Model) isSelectable(idx int) bool {
	vm := m.vms[idx]
	if !vm.Managed {
		return false
	}
	if vm.Section == "halted" && m.computeLayout().showHalted != haltedFull {
		return false
	}
	return true
}

// Helpers

func (m *Model) hasPendingAction(vmID string) bool {
	return m.pendingVMID != "" && m.pendingVMID == vmID
}

func (m *Model) selectedVM() *vmw.VM {
	if m.cursor >= 0 && m.cursor < len(m.vms) && m.isSelectable(m.cursor) {
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

// lastSelectableIndex returns the index of the last selectable VM.
func (m *Model) lastSelectableIndex() int {
	for i := len(m.vms) - 1; i >= 0; i-- {
		if m.isSelectable(i) {
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

func doPeek(ctx context.Context, client vmw.VMClient, vm *vmw.VM) tea.Cmd {
	return func() tea.Msg {
		raw, err := client.Peek(ctx, vm.ID)
		return peekMsg{vm.Name, raw, err}
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
