package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jaeyoung0509/letterpress/internal/domain"
	"github.com/jaeyoung0509/letterpress/internal/export"
	"github.com/jaeyoung0509/letterpress/internal/projectio"
	templatepkg "github.com/jaeyoung0509/letterpress/internal/template"
)

var (
	resolveTemplate = templatepkg.Resolve
	composeAndWrite = export.ComposeAndWrite
	saveProject     = projectio.Save
)

type Step string

const (
	StepQuickStart Step = "Quick Start"
	StepEdit       Step = "Edit Content"
	StepStyle      Step = "Style & Decorations"
	StepReview     Step = "Review & Export"
)

var stepOrder = []Step{
	StepQuickStart,
	StepEdit,
	StepStyle,
	StepReview,
}

type State struct {
	Current Step
}

func newState() State {
	return State{Current: StepQuickStart}
}

func (s State) currentIndex() int {
	for i, step := range stepOrder {
		if step == s.Current {
			return i
		}
	}
	return 0
}

func (s State) withNext() State {
	idx := s.currentIndex()
	if idx+1 < len(stepOrder) {
		s.Current = stepOrder[idx+1]
	}
	return s
}

func (s State) withPrev() State {
	idx := s.currentIndex()
	if idx > 0 {
		s.Current = stepOrder[idx-1]
	}
	return s
}

type KeyMap struct {
	Up        key.Binding
	Down      key.Binding
	NextFocus key.Binding
	PrevFocus key.Binding
	Select    key.Binding
	Back      key.Binding
	Quit      key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:        key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "move")),
		Down:      key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "move")),
		NextFocus: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field")),
		PrevFocus: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev field")),
		Select:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Back:      key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.NextFocus, k.Select, k.Back, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.NextFocus, k.PrevFocus},
		{k.Select, k.Back, k.Quit},
	}
}

type fileTargetKind string

const (
	fileTargetNone         fileTargetKind = ""
	fileTargetBodyImport   fileTargetKind = "body-import"
	fileTargetPrimaryImage fileTargetKind = "primary-image"
	fileTargetSlotImage    fileTargetKind = "slot-image"
)

const (
	focusQuickTemplateList  = "quick-template-list"
	focusQuickPageSize      = "quick-page-size"
	focusQuickOrientation   = "quick-orientation"
	focusQuickBodyImport    = "quick-body-import"
	focusQuickBodyBrowse    = "quick-body-browse"
	focusQuickPrimaryImage  = "quick-primary-image"
	focusQuickPrimaryBrowse = "quick-primary-browse"
	focusQuickExportFormat  = "quick-export-format"
	focusQuickOutputPath    = "quick-output-path"

	focusEditTitle     = "edit-title"
	focusEditBody      = "edit-body"
	focusEditSignature = "edit-signature"

	focusStyleDecorToggle = "style-decor-toggle"
	focusStyleDecorList   = "style-decor-list"

	focusReviewExport = "review-export"
	focusNavBack      = "nav-back"
	focusNavNext      = "nav-next"
)

type templateItem struct {
	entry TemplateEntry
}

func (i templateItem) Title() string       { return i.entry.Label() }
func (i templateItem) Description() string { return i.entry.Description() }
func (i templateItem) FilterValue() string { return i.entry.ID + " " + i.entry.Category }

type stepStatus struct {
	Text    string
	IsError bool
}

type Model struct {
	state       State
	composition CompositionState
	keyMap      KeyMap
	help        help.Model
	width       int
	height      int

	templates     []TemplateEntry
	templateList  list.Model
	templateIndex int
	focused       string

	titleInput        textinput.Model
	bodyImportInput   textinput.Model
	bodyInput         textarea.Model
	signatureInput    textinput.Model
	primaryImageInput textinput.Model
	advancedInputs    map[string]textinput.Model
	outputInput       textinput.Model

	decorationIndex int

	filePicker       filepicker.Model
	filePickerOpen   bool
	fileTargetKind   fileTargetKind
	fileTargetSlotID string

	statuses            map[Step]stepStatus
	suggestedOutputPath string
	customOutputPath    bool
}

func NewModel() Model {
	model := Model{
		state:          newState(),
		composition:    newCompositionState(),
		keyMap:         DefaultKeyMap(),
		help:           help.New(),
		templates:      loadTemplateEntries(),
		advancedInputs: map[string]textinput.Model{},
		statuses:       map[Step]stepStatus{},
		width:          120,
		height:         40,
	}

	model.templateList = newTemplateList(model.templates)
	model.titleInput = newTextInput("Title", "Add a short heading")
	model.bodyImportInput = newTextInput("Body import", "Paste a .txt or .md file path")
	model.bodyInput = newBodyTextarea()
	model.signatureInput = newTextInput("Signature", "Add a closing name")
	model.primaryImageInput = newTextInput("Primary image", "Paste an image path or browse")
	model.outputInput = newTextInput("Output", "exports/letterpress.pdf")

	if len(model.templates) > 0 {
		model = model.selectTemplate(0)
	} else {
		model.syncSuggestedOutputPath(true)
	}

	model.setFocused(model.defaultFocusForStep(model.state.Current))
	model.resizeComponents()
	return model
}

func Run() error {
	program := tea.NewProgram(NewModel(), tea.WithAltScreen())
	_, err := program.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeComponents()
		if m.filePickerOpen {
			m.filePicker.SetHeight(m.modalHeight())
		}
	}

	if m.filePickerOpen {
		return m.updateFilePicker(msg)
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, m.keyMap.Quit):
			return m, tea.Quit
		case key.Matches(keyMsg, m.keyMap.PrevFocus):
			m.focusPrev()
			return m, nil
		case key.Matches(keyMsg, m.keyMap.NextFocus):
			m.focusNext()
			return m, nil
		case key.Matches(keyMsg, m.keyMap.Back):
			m.movePrevStep()
			return m, nil
		case key.Matches(keyMsg, m.keyMap.Select):
			return m.activateFocused()
		}
	}

	var cmd tea.Cmd
	switch m.state.Current {
	case StepQuickStart:
		m, cmd = m.updateQuickStartStep(msg)
	case StepEdit:
		m, cmd = m.updateEditStep(msg)
	case StepStyle:
		m, cmd = m.updateStyleStep(msg)
	case StepReview:
		m, cmd = m.updateReviewStep(msg)
	}

	return m, cmd
}

func (m Model) updateQuickStartStep(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.focused {
	case focusQuickTemplateList:
		next, listCmd := m.templateList.Update(msg)
		m.templateList = next
		m = m.syncSelectedTemplateFromList()
		cmd = listCmd
	case focusQuickBodyImport:
		next, inputCmd := m.bodyImportInput.Update(msg)
		m.bodyImportInput = next
		cmd = inputCmd
	case focusQuickPrimaryImage:
		next, inputCmd := m.primaryImageInput.Update(msg)
		m.primaryImageInput = next
		cmd = inputCmd
	case focusQuickOutputPath:
		next, inputCmd := m.outputInput.Update(msg)
		m.outputInput = next
		m.updateOutputPathFromInput()
		cmd = inputCmd
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch m.focused {
		case focusQuickPageSize:
			if key.Matches(keyMsg, m.keyMap.Up) {
				m = m.cycleSize(-1)
			} else if key.Matches(keyMsg, m.keyMap.Down) {
				m = m.cycleSize(1)
			}
		case focusQuickOrientation:
			if key.Matches(keyMsg, m.keyMap.Up) || key.Matches(keyMsg, m.keyMap.Down) {
				m = m.toggleOrientation()
			}
		case focusQuickExportFormat:
			if key.Matches(keyMsg, m.keyMap.Up) || key.Matches(keyMsg, m.keyMap.Down) {
				m = m.toggleExportFormat(StepQuickStart)
			}
		}
	}

	return m, cmd
}

func (m Model) updateEditStep(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.focused {
	case focusEditTitle:
		next, inputCmd := m.titleInput.Update(msg)
		m.titleInput = next
		m.composition.Project.Content.Title = m.titleInput.Value()
		cmd = inputCmd
	case focusEditBody:
		next, inputCmd := m.bodyInput.Update(msg)
		m.bodyInput = next
		m.composition.Project.Content.Body = m.bodyInput.Value()
		cmd = inputCmd
	case focusEditSignature:
		next, inputCmd := m.signatureInput.Update(msg)
		m.signatureInput = next
		m.composition.Project.Content.Signature = m.signatureInput.Value()
		cmd = inputCmd
	}

	return m, cmd
}

func (m Model) updateStyleStep(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	if slotID, ok := styleSlotIDFromFocus(m.focused); ok {
		input := m.advancedInputs[slotID]
		next, inputCmd := input.Update(msg)
		input = next
		m.advancedInputs[slotID] = input
		cmd = inputCmd
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok && m.focused == focusStyleDecorList {
		assets := m.currentTemplateDecorationAssets()
		if len(assets) > 0 {
			if key.Matches(keyMsg, m.keyMap.Up) {
				m.decorationIndex--
				if m.decorationIndex < 0 {
					m.decorationIndex = len(assets) - 1
				}
			} else if key.Matches(keyMsg, m.keyMap.Down) {
				m.decorationIndex = (m.decorationIndex + 1) % len(assets)
			}
		}
	}

	return m, cmd
}

func (m Model) updateReviewStep(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

func (m Model) updateFilePicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, m.keyMap.Quit) {
			return m, tea.Quit
		}
		if key.Matches(keyMsg, m.keyMap.Back) {
			m.filePickerOpen = false
			m.fileTargetKind = fileTargetNone
			m.fileTargetSlotID = ""
			return m, nil
		}
	}

	next, cmd := m.filePicker.Update(msg)
	m.filePicker = next

	if ok, path := m.filePicker.DidSelectDisabledFile(msg); ok {
		m.setStatus(m.state.Current, fmt.Sprintf("file type not supported here: %s", path), true)
		return m, cmd
	}

	if ok, path := m.filePicker.DidSelectFile(msg); ok {
		m.applyFileSelection(path)
		m.filePickerOpen = false
		m.fileTargetKind = fileTargetNone
		m.fileTargetSlotID = ""
		return m, nil
	}

	return m, cmd
}

func (m Model) activateFocused() (tea.Model, tea.Cmd) {
	switch m.focused {
	case focusNavBack:
		m.movePrevStep()
	case focusNavNext:
		m.moveNextStep()
	case focusQuickTemplateList:
		m.focusNext()
	case focusQuickPageSize:
		m = m.cycleSize(1)
	case focusQuickOrientation:
		m = m.toggleOrientation()
	case focusQuickBodyImport:
		if strings.TrimSpace(m.bodyImportInput.Value()) == "" {
			return m.openFilePicker(fileTargetBodyImport, "")
		}
		m = m.applyBodyImport()
	case focusQuickBodyBrowse:
		return m.openFilePicker(fileTargetBodyImport, "")
	case focusQuickPrimaryImage:
		if strings.TrimSpace(m.primaryImageInput.Value()) == "" {
			return m.openFilePicker(fileTargetPrimaryImage, "")
		}
		m = m.applyPrimaryImageValue(m.primaryImageInput.Value(), StepQuickStart)
	case focusQuickPrimaryBrowse:
		return m.openFilePicker(fileTargetPrimaryImage, "")
	case focusQuickExportFormat:
		m = m.toggleExportFormat(StepQuickStart)
	case focusQuickOutputPath:
		if strings.TrimSpace(m.outputInput.Value()) == "" {
			m.syncSuggestedOutputPath(true)
			m.setStatus(StepQuickStart, "Recommended output path restored.", false)
		} else {
			m.setStatus(StepQuickStart, "Output path updated.", false)
		}
	case focusStyleDecorToggle:
		m = m.toggleDecorationsEnabled()
	case focusStyleDecorList:
		m = m.toggleCurrentDecoration()
	case focusReviewExport:
		m = m.exportComposition()
	default:
		if slotID, ok := styleSlotIDFromFocus(m.focused); ok {
			input := m.advancedInputs[slotID]
			if strings.TrimSpace(input.Value()) == "" {
				return m.openFilePicker(fileTargetSlotImage, slotID)
			}
			m = m.applyImageValue(slotID, input.Value(), StepStyle)
		}
		if slotID, ok := styleSlotBrowseIDFromFocus(m.focused); ok {
			return m.openFilePicker(fileTargetSlotImage, slotID)
		}
	}

	return m, nil
}

func (m *Model) moveNextStep() {
	previous := m.state.Current
	m.state = m.state.withNext()
	if m.state.Current != previous {
		m.setFocused(m.defaultFocusForStep(m.state.Current))
	}
}

func (m *Model) movePrevStep() {
	previous := m.state.Current
	m.state = m.state.withPrev()
	if m.state.Current != previous {
		m.setFocused(m.defaultFocusForStep(m.state.Current))
	}
}

func (m *Model) focusNext() {
	focusables := m.focusables()
	if len(focusables) == 0 {
		return
	}
	idx := slices.Index(focusables, m.focused)
	if idx == -1 {
		m.setFocused(focusables[0])
		return
	}
	m.setFocused(focusables[(idx+1)%len(focusables)])
}

func (m *Model) focusPrev() {
	focusables := m.focusables()
	if len(focusables) == 0 {
		return
	}
	idx := slices.Index(focusables, m.focused)
	if idx == -1 {
		m.setFocused(focusables[0])
		return
	}
	idx--
	if idx < 0 {
		idx = len(focusables) - 1
	}
	m.setFocused(focusables[idx])
}

func (m *Model) setFocused(id string) {
	m.focused = id
	m.syncFocus()
}

func (m *Model) syncFocus() {
	m.titleInput.Blur()
	m.bodyImportInput.Blur()
	m.bodyInput.Blur()
	m.signatureInput.Blur()
	m.primaryImageInput.Blur()
	m.outputInput.Blur()
	for slotID, input := range m.advancedInputs {
		input.Blur()
		m.advancedInputs[slotID] = input
	}

	switch m.focused {
	case focusQuickBodyImport:
		m.bodyImportInput.Focus()
	case focusQuickPrimaryImage:
		m.primaryImageInput.Focus()
	case focusQuickOutputPath:
		m.outputInput.Focus()
	case focusEditTitle:
		m.titleInput.Focus()
	case focusEditBody:
		m.bodyInput.Focus()
	case focusEditSignature:
		m.signatureInput.Focus()
	default:
		if slotID, ok := styleSlotIDFromFocus(m.focused); ok {
			input := m.advancedInputs[slotID]
			input.Focus()
			m.advancedInputs[slotID] = input
		}
	}
}

func (m Model) focusables() []string {
	switch m.state.Current {
	case StepQuickStart:
		return []string{
			focusQuickTemplateList,
			focusQuickPageSize,
			focusQuickOrientation,
			focusQuickBodyImport,
			focusQuickBodyBrowse,
			focusQuickPrimaryImage,
			focusQuickPrimaryBrowse,
			focusQuickExportFormat,
			focusQuickOutputPath,
			focusNavNext,
		}
	case StepEdit:
		return []string{
			focusEditTitle,
			focusEditBody,
			focusEditSignature,
			focusNavBack,
			focusNavNext,
		}
	case StepStyle:
		focuses := []string{focusStyleDecorToggle}
		if len(m.currentTemplateDecorationAssets()) > 0 {
			focuses = append(focuses, focusStyleDecorList)
		}
		for _, slot := range m.additionalImageSlots() {
			focuses = append(focuses, styleSlotFocus(slot.ID), styleSlotBrowseFocus(slot.ID))
		}
		focuses = append(focuses, focusNavBack, focusNavNext)
		return focuses
	case StepReview:
		return []string{focusReviewExport, focusNavBack}
	default:
		return nil
	}
}

func (m Model) defaultFocusForStep(step Step) string {
	switch step {
	case StepQuickStart:
		return focusQuickTemplateList
	case StepEdit:
		return focusEditTitle
	case StepStyle:
		if len(m.currentTemplateDecorationAssets()) > 0 {
			return focusStyleDecorToggle
		}
		if extra := m.additionalImageSlots(); len(extra) > 0 {
			return styleSlotFocus(extra[0].ID)
		}
		return focusNavNext
	case StepReview:
		return focusReviewExport
	default:
		return focusQuickTemplateList
	}
}

func (m *Model) resizeComponents() {
	mainWidth := m.mainPanelWidth()
	m.titleInput.Width = max(20, mainWidth/2-8)
	m.bodyImportInput.Width = max(28, mainWidth/2-16)
	m.signatureInput.Width = max(20, mainWidth/2-8)
	m.primaryImageInput.Width = max(28, mainWidth/2-16)
	m.outputInput.Width = max(24, mainWidth/2-10)
	m.bodyInput.SetWidth(max(36, mainWidth-10))
	m.bodyInput.SetHeight(min(12, max(8, m.height-24)))

	for slotID, input := range m.advancedInputs {
		input.Width = max(28, mainWidth/2-16)
		m.advancedInputs[slotID] = input
	}
}

func newTemplateList(entries []TemplateEntry) list.Model {
	items := make([]list.Item, 0, len(entries))
	for _, entry := range entries {
		items = append(items, templateItem{entry: entry})
	}

	delegate := list.NewDefaultDelegate()
	model := list.New(items, delegate, 0, 0)
	model.SetShowTitle(false)
	model.SetShowFilter(false)
	model.SetShowHelp(false)
	model.SetShowPagination(false)
	model.SetShowStatusBar(false)
	model.SetFilteringEnabled(false)
	return model
}

func newTextInput(prompt, placeholder string) textinput.Model {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = placeholder
	input.CharLimit = 512
	input.Width = 48
	input.SetValue("")
	_ = prompt
	return input
}

func newBodyTextarea() textarea.Model {
	input := textarea.New()
	input.Prompt = ""
	input.Placeholder = "Write the body here, or import it during Quick Start."
	input.ShowLineNumbers = false
	input.SetHeight(10)
	input.SetWidth(60)
	return input
}

func (m Model) View() string {
	m.resizeComponents()

	header := strings.Join([]string{
		headerStyle.Render("letterpress"),
		subtitleStyle.Render("Quick Start first, then fine-tune content and style."),
		"",
		m.renderSteps(),
	}, "\n")

	var body string
	if m.state.Current == StepQuickStart {
		body = panelStyle.Width(m.fullPanelWidth()).Render(m.renderQuickStartStep())
	} else {
		main := panelStyle.Width(m.mainPanelWidth()).Render(m.renderCurrentStep())
		sidebar := summaryPanelStyle.Width(m.summaryWidth()).Render(m.renderSidebar())
		body = joinPanels(main, sidebar, m.width)
	}

	footer := mutedStyle.Render(m.help.View(m.keyMap))
	view := strings.Join([]string{
		header,
		"",
		body,
		"",
		footer,
	}, "\n")

	if m.filePickerOpen {
		modal := modalStyle.Width(min(m.mainPanelWidth(), m.width-8)).Render(m.renderFilePickerModal())
		view = strings.Join([]string{view, "", modal}, "\n")
	}

	return appFrameStyle.Width(max(60, m.width)).Render(view)
}

func (m Model) renderCurrentStep() string {
	switch m.state.Current {
	case StepEdit:
		return m.renderEditStep()
	case StepStyle:
		return m.renderStyleStep()
	case StepReview:
		return m.renderReviewStep()
	default:
		return m.renderQuickStartStep()
	}
}

func (m Model) renderSteps() string {
	chips := make([]string, 0, len(stepOrder))
	for _, step := range stepOrder {
		style := stepChipStyle
		if step == m.state.Current {
			style = activeStepChipStyle
		}
		chips = append(chips, style.Render(string(step)))
	}
	return strings.Join(chips, " ")
}

func (m Model) renderQuickStartStep() string {
	contentWidth := max(60, m.fullPanelWidth()-8)
	entry, _ := m.currentTemplateEntry()
	stack := m.width < 155
	leftWidth := contentWidth
	rightWidth := 0
	if !stack {
		leftWidth = max(58, int(float64(contentWidth)*0.57))
		rightWidth = max(34, contentWidth-leftWidth-4)
	}

	header := strings.Join([]string{
		sectionTitleStyle.Render("1. Quick Start"),
		mutedStyle.Render("Pick the template, bring in text and an image, then export with the suggested defaults."),
	}, "\n")

	top := joinPanels(
		m.renderQuickTemplateBlock(entry, leftWidth),
		m.renderLivePreviewCard(entry, rightWidth),
		contentWidth,
	)

	bottom := joinPanels(
		strings.Join([]string{
			m.renderQuickPageBlock(entry),
			"",
			m.renderQuickBodyImportBlock(),
		}, "\n"),
		strings.Join([]string{
			m.renderQuickPrimaryImageBlock(entry),
			"",
			m.renderQuickOutputBlock(),
			"",
			m.renderQuickReadiness(entry),
		}, "\n"),
		contentWidth,
	)

	parts := []string{
		header,
		"",
		top,
		"",
		bottom,
		renderStatusBlock(m.status(StepQuickStart)),
		"",
		m.renderNavigation(false, true),
	}

	if stack {
		return strings.Join(parts, "\n")
	}

	return strings.Join(parts, "\n")
}

func (m Model) renderQuickTemplateBlock(entry TemplateEntry, width int) string {
	cardWidth := max(26, min(32, width/2-3))
	cards := make([]string, 0, len(m.templates))
	for idx, candidate := range m.templates {
		selected := idx == m.templateIndex
		cards = append(cards, m.renderTemplateCard(candidate, cardWidth, selected, m.focused == focusQuickTemplateList && selected))
	}

	lines := []string{
		labelStyle.Render("Templates"),
		mutedStyle.Render("Use Up/Down to move between cards, then Tab into the rest of the setup."),
		"",
	}

	if width < 76 {
		lines = append(lines, strings.Join(cards, "\n\n"))
	} else {
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top, cards...))
	}

	lines = append(lines,
		"",
		eyebrowStyle.Render("Selected default"),
		fmt.Sprintf("%s · %s · %s", entry.ID, entry.Category, formatSizes(entry.SupportedSizes)),
	)

	return strings.Join(lines, "\n")
}

func (m Model) renderQuickPageBlock(entry TemplateEntry) string {
	body := strings.Join([]string{
		labelStyle.Render("Page"),
		m.renderOptionGroup("Paper size", entry.SupportedSizes, string(m.composition.Project.Page.Size), m.focused == focusQuickPageSize),
		"",
		m.renderOrientationGroup(m.focused == focusQuickOrientation),
	}, "\n")
	return subCardStyle.Width(max(30, m.fullPanelWidth()/2-10)).Render(body)
}

func (m Model) renderQuickBodyImportBlock() string {
	body := strings.Join([]string{
		labelStyle.Render("Text file (.txt / .md)"),
		m.bodyImportInput.View() + " " + m.renderButton("Browse", m.focused == focusQuickBodyBrowse, false),
		mutedStyle.Render("Press Enter in the field to apply the typed path into the body."),
	}, "\n")
	if m.focused == focusQuickBodyImport || m.focused == focusQuickBodyBrowse {
		return subCardFocusedStyle.Width(max(30, m.fullPanelWidth()/2-10)).Render(body)
	}
	return subCardStyle.Width(max(30, m.fullPanelWidth()/2-10)).Render(body)
}

func (m Model) renderQuickPrimaryImageBlock(entry TemplateEntry) string {
	label := "Primary image"
	if slot, ok := entry.PrimaryImageSlot(); ok {
		label = "Primary image (" + slot.ID + ")"
	}

	body := strings.Join([]string{
		labelStyle.Render(label),
		m.primaryImageInput.View() + " " + m.renderButton("Browse", m.focused == focusQuickPrimaryBrowse, false),
		mutedStyle.Render("Press Enter in the field to bind the current path."),
	}, "\n")
	if m.focused == focusQuickPrimaryImage || m.focused == focusQuickPrimaryBrowse {
		return subCardFocusedStyle.Width(max(30, m.fullPanelWidth()/2-10)).Render(body)
	}
	return subCardStyle.Width(max(30, m.fullPanelWidth()/2-10)).Render(body)
}

func (m Model) renderQuickOutputBlock() string {
	body := strings.Join([]string{
		labelStyle.Render("Output"),
		m.renderFormatSelector(m.focused == focusQuickExportFormat),
		"",
		m.renderFieldBlock("Output path", m.outputInput.View(), m.focused == focusQuickOutputPath),
		mutedStyle.Render("Recommended: " + m.suggestedOutputPath),
	}, "\n")
	return subCardStyle.Width(max(30, m.fullPanelWidth()/2-10)).Render(body)
}

func (m Model) renderQuickReadiness(entry TemplateEntry) string {
	bodyState := "optional, edit later"
	if path := strings.TrimSpace(m.composition.BodyImportPath); path != "" {
		bodyState = "imported from " + summarizeText(path)
	}

	imageState := "optional"
	if slot, ok := entry.PrimaryImageSlot(); ok {
		if path := strings.TrimSpace(m.composition.ImagePath(slot.ID)); path != "" {
			imageState = "set to " + summarizeText(path)
		}
	}

	outputState := "missing"
	if path := strings.TrimSpace(m.composition.Project.Export.Out); path != "" {
		outputState = "ready"
	}

	body := strings.Join([]string{
		labelStyle.Render("Ready Check"),
		"Template: selected",
		fmt.Sprintf("Page: %s %s", m.composition.Project.Page.Size, m.composition.Project.Page.Orientation),
		"Body: " + bodyState,
		"Primary image: " + imageState,
		"Output: " + outputState,
	}, "\n")
	return subCardStyle.Width(max(30, m.fullPanelWidth()/2-10)).Render(body)
}

func (m Model) renderTemplateCard(entry TemplateEntry, width int, selected, focused bool) string {
	style := templateCardStyle.Width(width)
	switch {
	case selected && focused:
		style = templateCardFocusedStyle.Width(width)
	case selected:
		style = templateCardSelectedStyle.Width(width)
	case focused:
		style = subCardFocusedStyle.Width(width)
	}

	features := []string{"TEXT"}
	if len(entry.ImageSlots) > 0 {
		features = append(features, fmt.Sprintf("IMG %d", len(entry.ImageSlots)))
	}
	if len(entry.DecorationAssets) > 0 {
		features = append(features, fmt.Sprintf("DECOR %d", len(entry.DecorationAssets)))
	}

	lines := []string{
		eyebrowStyle.Render(strings.ToUpper(entry.Category)),
		labelStyle.Render(entry.ID),
		mutedStyle.Render("Sizes: " + formatSizes(entry.SupportedSizes)),
		mutedStyle.Render("Features: " + strings.Join(features, " · ")),
	}
	if selected {
		lines = append(lines, successStyle.Render("Selected"))
	}

	return style.Render(strings.Join(lines, "\n"))
}

func (m Model) renderLivePreviewCard(entry TemplateEntry, width int) string {
	if width <= 0 {
		width = max(34, m.fullPanelWidth()/2-10)
	}

	previewWidth := max(28, width-4)
	lines := []string{
		eyebrowStyle.Render("LIVE PREVIEW"),
		labelStyle.Render(entry.ID),
		mutedStyle.Render(fmt.Sprintf("%s %s", m.composition.Project.Page.Size, m.composition.Project.Page.Orientation)),
		"",
		pagePreviewStyle.Width(previewWidth).Render(m.renderPreviewPage(entry, previewWidth-4)),
		"",
		mutedStyle.Render("This is schematic. Export output stays the source of truth."),
	}

	return previewFrameStyle.Width(max(32, width)).Render(strings.Join(lines, "\n"))
}

func (m Model) renderPreviewPage(entry TemplateEntry, width int) string {
	innerWidth := max(20, width)
	rows := make([]string, 0, 12)

	if m.composition.Project.Options.Decorations && len(entry.DecorationAssets) > 0 {
		rows = append(rows, centerPreview("ornaments enabled", innerWidth))
	}

	if previewHasTextRole(entry, "title", "heading", "headline", "greeting") {
		rows = append(rows, alignPreview(previewPrimaryText(entry, m.composition.Project.Content.Title, "Title"), innerWidth, "center"))
		rows = append(rows, strings.Repeat("─", max(10, innerWidth-2)))
	}

	if len(entry.ImageSlots) > 0 {
		rows = append(rows, previewImageBox(m.previewImageLabel(entry), innerWidth))
	}

	if previewHasTextRole(entry, "body", "message", "note") {
		body := previewPrimaryText(entry, m.composition.Project.Content.Body, "Body text")
		rows = append(rows, wrapPreviewText(body, innerWidth, 4)...)
	}

	if previewHasTextRole(entry, "signature", "signoff", "closing") {
		rows = append(rows, "")
		rows = append(rows, alignPreview(previewPrimaryText(entry, m.composition.Project.Content.Signature, "Signature"), innerWidth, "right"))
	}

	targetHeight := 12
	if m.composition.Project.Page.Orientation == domain.OrientationLandscape {
		targetHeight = 9
	}
	for len(rows) < targetHeight {
		rows = append(rows, "")
	}

	return strings.Join(rows[:targetHeight], "\n")
}

func (m Model) previewImageLabel(entry TemplateEntry) string {
	slot, ok := entry.PrimaryImageSlot()
	if !ok {
		return "No image"
	}
	path := strings.TrimSpace(m.composition.ImagePath(slot.ID))
	if path == "" {
		return "Add image"
	}
	return filepath.Base(path)
}

func previewHasTextRole(entry TemplateEntry, roles ...string) bool {
	for _, slot := range entry.Template.Slots {
		if slot.Type != domain.SlotTypeText {
			continue
		}
		id := strings.ToLower(strings.TrimSpace(slot.ID))
		for _, role := range roles {
			if id == role {
				return true
			}
		}
	}
	return false
}

func previewPrimaryText(entry TemplateEntry, value, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return fallback
}

func previewImageBox(label string, width int) string {
	boxWidth := max(12, width-2)
	top := "┌" + strings.Repeat("─", boxWidth-2) + "┐"
	middle := "│" + padPreview(centerPreview("Image", boxWidth-2), boxWidth-2) + "│"
	bottom := "└" + strings.Repeat("─", boxWidth-2) + "┘"
	caption := centerPreview(label, width)
	return strings.Join([]string{top, middle, bottom, caption}, "\n")
}

func wrapPreviewText(text string, width, lines int) []string {
	words := strings.Fields(strings.TrimSpace(text))
	if len(words) == 0 {
		return []string{mutedStyle.Render("(empty body)")}
	}

	out := make([]string, 0, lines)
	current := ""
	for _, word := range words {
		candidate := word
		if current != "" {
			candidate = current + " " + word
		}
		if len(candidate) > width && current != "" {
			out = append(out, current)
			current = word
			if len(out) == lines {
				break
			}
			continue
		}
		current = candidate
	}
	if len(out) < lines && current != "" {
		out = append(out, current)
	}
	if len(out) > lines {
		out = out[:lines]
	}
	if len(out) == lines && len(words) > 0 {
		last := out[len(out)-1]
		if len(last) > width-1 {
			last = last[:max(0, width-1)]
		}
		if !strings.HasSuffix(last, "...") && len(strings.Fields(text)) > len(strings.Fields(strings.Join(out, " "))) {
			last = strings.TrimRight(last, " ") + "..."
		}
		out[len(out)-1] = last
	}
	return out
}

func alignPreview(text string, width int, mode string) string {
	text = summarizeTextForWidth(text, width)
	padding := max(0, width-len(text))
	switch mode {
	case "center":
		left := padding / 2
		right := padding - left
		return strings.Repeat(" ", left) + text + strings.Repeat(" ", right)
	case "right":
		return strings.Repeat(" ", padding) + text
	default:
		return text + strings.Repeat(" ", padding)
	}
}

func centerPreview(text string, width int) string {
	return alignPreview(text, width, "center")
}

func padPreview(text string, width int) string {
	text = summarizeTextForWidth(text, width)
	if len(text) < width {
		return text + strings.Repeat(" ", width-len(text))
	}
	return text
}

func summarizeTextForWidth(text string, width int) string {
	text = strings.TrimSpace(text)
	if width <= 0 {
		return ""
	}
	if len(text) <= width {
		return text
	}
	if width <= 3 {
		return text[:width]
	}
	return text[:width-3] + "..."
}

func (m Model) renderEditStep() string {
	header := []string{
		sectionTitleStyle.Render("2. Edit Content"),
		mutedStyle.Render("Fine-tune title, body, and signature after the quick setup is done."),
	}

	top := joinPanels(
		m.renderFieldBlock("Title", m.titleInput.View(), m.focused == focusEditTitle),
		m.renderFieldBlock("Signature", m.signatureInput.View(), m.focused == focusEditSignature),
		m.mainPanelWidth(),
	)

	parts := []string{
		strings.Join(header, "\n"),
		"",
		top,
		"",
		m.renderFieldBlock("Body", m.bodyInput.View(), m.focused == focusEditBody),
		"",
		m.renderNavigation(true, true),
	}

	return strings.Join(parts, "\n")
}

func (m Model) renderStyleStep() string {
	header := []string{
		sectionTitleStyle.Render("3. Style & Decorations"),
		mutedStyle.Render("Turn decorations on or off and fill extra image slots only if the template needs them."),
	}

	left := m.renderStyleDecorationsBlock()
	right := m.renderStyleAdvancedImagesBlock()

	content := joinPanels(left, right, m.mainPanelWidth())
	if right == "" {
		content = left
	}

	parts := []string{
		strings.Join(header, "\n"),
		"",
		content,
		renderStatusBlock(m.status(StepStyle)),
		"",
		m.renderNavigation(true, true),
	}

	return strings.Join(parts, "\n")
}

func (m Model) renderStyleDecorationsBlock() string {
	assets := m.currentTemplateDecorationAssets()
	lines := []string{
		labelStyle.Render("Decorations"),
		m.renderButton(m.decorationsToggleLabel(), m.focused == focusStyleDecorToggle, false),
	}

	if len(assets) == 0 {
		lines = append(lines, mutedStyle.Render("No decoration assets in this template."))
	} else {
		for idx, asset := range assets {
			prefix := "  "
			if idx == m.decorationIndex {
				prefix = "→ "
			}
			state := "[ ]"
			if m.composition.DecorationEnabled(asset.ID) {
				state = "[x]"
			}
			line := fmt.Sprintf("%s%s %s", prefix, state, asset.ID)
			if m.focused == focusStyleDecorList && idx == m.decorationIndex {
				line = focusedBlockStyle.Render(line)
			}
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderStyleAdvancedImagesBlock() string {
	extra := m.additionalImageSlots()
	if len(extra) == 0 {
		return ""
	}

	lines := []string{
		labelStyle.Render("Advanced image slots"),
		mutedStyle.Render("Only fill these if the selected template has more than one image area."),
	}

	for _, slot := range extra {
		input := m.advancedInputs[slot.ID]
		block := strings.Join([]string{
			labelStyle.Render(slot.ID),
			input.View() + " " + m.renderButton("Browse", m.focused == styleSlotBrowseFocus(slot.ID), false),
			mutedStyle.Render("Press Enter in the field to bind the current path."),
		}, "\n")

		if m.focused == styleSlotFocus(slot.ID) || m.focused == styleSlotBrowseFocus(slot.ID) {
			block = focusedBlockStyle.Render(block)
		}

		lines = append(lines, block)
	}

	return strings.Join(lines, "\n\n")
}

func (m Model) renderReviewStep() string {
	left := strings.Join([]string{
		sectionTitleStyle.Render("4. Review & Export"),
		mutedStyle.Render("Check the final configuration, then export from here."),
		"",
		labelStyle.Render("Composition"),
		m.renderReviewSummary(),
		renderStatusBlock(m.status(StepReview)),
	}, "\n")

	right := strings.Join([]string{
		labelStyle.Render("Export target"),
		fmt.Sprintf("Format: %s", strings.ToUpper(string(m.composition.Project.Export.Format))),
		fmt.Sprintf("Output: %s", summarizeText(m.composition.Project.Export.Out)),
		fmt.Sprintf("Project file: %s", summarizeText(m.exportProjectPath())),
		"",
		m.renderButton("Export now", m.focused == focusReviewExport, true),
		"",
		m.renderNavigation(true, false),
	}, "\n")

	return joinPanels(left, right, m.mainPanelWidth())
}

func (m Model) renderSidebar() string {
	entry, _ := m.currentTemplateEntry()
	blocks := []string{
		m.renderLivePreviewCard(entry, max(30, m.summaryWidth()-8)),
		m.renderSummary(),
	}

	switch m.state.Current {
	case StepEdit:
		blocks = append(blocks, strings.Join([]string{
			labelStyle.Render("Editing Tip"),
			"Use Quick Start for file import.",
			"Use this step for polish only.",
		}, "\n"))
	case StepStyle:
		blocks = append(blocks, strings.Join([]string{
			labelStyle.Render("Style Tip"),
			fmt.Sprintf("%d decorations available", len(m.currentTemplateDecorationAssets())),
			fmt.Sprintf("%d extra image slots", len(m.additionalImageSlots())),
		}, "\n"))
	case StepReview:
		blocks = append(blocks, strings.Join([]string{
			labelStyle.Render("Final Check"),
			"Output path is set",
			"Export format is chosen",
			"Project YAML is saved on export",
		}, "\n"))
	}

	return strings.Join(blocks, "\n\n")
}

func (m Model) renderSummary() string {
	entry, _ := m.currentTemplateEntry()

	primary := "Not set"
	if slot, ok := entry.PrimaryImageSlot(); ok {
		if path := strings.TrimSpace(m.composition.ImagePath(slot.ID)); path != "" {
			primary = summarizeText(path)
		}
	}

	lines := []string{
		sectionTitleStyle.Render("Session"),
		"",
		labelStyle.Render("Template"),
		entry.ID,
		labelStyle.Render("Page"),
		fmt.Sprintf("%s %s", m.composition.Project.Page.Size, m.composition.Project.Page.Orientation),
		labelStyle.Render("Body import"),
		m.bodyImportSummary(),
		labelStyle.Render("Primary image"),
		primary,
		labelStyle.Render("Decorations"),
		fmt.Sprintf("%d selected", m.composition.DecorationCount()),
		labelStyle.Render("Export"),
		fmt.Sprintf("%s -> %s", strings.ToUpper(string(m.composition.Project.Export.Format)), summarizeText(m.composition.Project.Export.Out)),
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderReviewSummary() string {
	lines := []string{
		fmt.Sprintf("Template: %s", m.composition.Project.Template),
		fmt.Sprintf("Page: %s %s", m.composition.Project.Page.Size, m.composition.Project.Page.Orientation),
		fmt.Sprintf("Title: %s", summarizeText(m.composition.Project.Content.Title)),
		fmt.Sprintf("Body: %s", summarizeText(m.composition.Project.Content.Body)),
		fmt.Sprintf("Signature: %s", summarizeText(m.composition.Project.Content.Signature)),
	}

	entry, _ := m.currentTemplateEntry()
	if slot, ok := entry.PrimaryImageSlot(); ok {
		lines = append(lines, fmt.Sprintf("Primary image: %s", summarizeText(m.composition.ImagePath(slot.ID))))
	}
	for _, slot := range m.additionalImageSlots() {
		lines = append(lines, fmt.Sprintf("%s: %s", slot.ID, summarizeText(m.composition.ImagePath(slot.ID))))
	}
	lines = append(lines, fmt.Sprintf("Decorations: %d selected", m.composition.DecorationCount()))
	return strings.Join(lines, "\n")
}

func (m Model) renderFieldBlock(label, view string, focused bool) string {
	block := strings.Join([]string{labelStyle.Render(label), view}, "\n")
	if focused {
		return focusedBlockStyle.Render(block)
	}
	return block
}

func (m Model) renderOptionGroup(label string, sizes []domain.PageSize, selected string, focused bool) string {
	values := make([]string, 0, len(sizes))
	for _, size := range sizes {
		values = append(values, m.renderChoice(string(size), selected == string(size), focused))
	}

	block := strings.Join([]string{
		labelStyle.Render(label),
		strings.Join(values, " "),
	}, "\n")
	if focused {
		return focusedBlockStyle.Render(block)
	}
	return block
}

func (m Model) renderOrientationGroup(focused bool) string {
	block := strings.Join([]string{
		labelStyle.Render("Orientation"),
		strings.Join([]string{
			m.renderChoice("portrait", m.composition.Project.Page.Orientation == domain.OrientationPortrait, focused),
			m.renderChoice("landscape", m.composition.Project.Page.Orientation == domain.OrientationLandscape, focused),
		}, " "),
	}, "\n")
	if focused {
		return focusedBlockStyle.Render(block)
	}
	return block
}

func (m Model) renderFormatSelector(focused bool) string {
	block := strings.Join([]string{
		labelStyle.Render("Format"),
		strings.Join([]string{
			m.renderChoice("PDF", m.composition.Project.Export.Format == domain.ExportFormatPDF, focused),
			m.renderChoice("PNG", m.composition.Project.Export.Format == domain.ExportFormatPNG, focused),
		}, " "),
	}, "\n")
	if focused {
		return focusedBlockStyle.Render(block)
	}
	return block
}

func (m Model) renderChoice(label string, selected bool, focused bool) string {
	style := buttonStyle
	if selected {
		style = buttonPrimaryStyle
	}
	if focused && selected {
		style = buttonPrimaryFocusedStyle
	} else if focused {
		style = buttonFocusedStyle
	}
	return style.Render(label)
}

func (m Model) renderButton(label string, focused, primary bool) string {
	style := buttonStyle
	if primary {
		style = buttonPrimaryStyle
	}
	if focused && primary {
		style = buttonPrimaryFocusedStyle
	} else if focused {
		style = buttonFocusedStyle
	}
	return style.Render(label)
}

func (m Model) renderNavigation(showBack, showNext bool) string {
	buttons := make([]string, 0, 2)
	if showBack {
		buttons = append(buttons, m.renderButton("Back", m.focused == focusNavBack, false))
	}
	if showNext {
		buttons = append(buttons, m.renderButton("Continue", m.focused == focusNavNext, true))
	}
	return strings.Join(buttons, " ")
}

func renderStatusBlock(status stepStatus) string {
	if status.Text == "" {
		return ""
	}
	if status.IsError {
		return "\n" + errorStyle.Render(status.Text)
	}
	return "\n" + successStyle.Render(status.Text)
}

func (m Model) renderFilePickerModal() string {
	target := "file"
	switch m.fileTargetKind {
	case fileTargetBodyImport:
		target = "body import (.txt, .md)"
	case fileTargetPrimaryImage, fileTargetSlotImage:
		target = "image (.png, .jpg, .jpeg, .webp)"
	}

	return strings.Join([]string{
		sectionTitleStyle.Render("Browse local files"),
		mutedStyle.Render("Enter selects a file. Backspace moves up a directory. Esc closes the modal."),
		"",
		labelStyle.Render("Current directory"),
		m.filePicker.CurrentDirectory,
		"",
		labelStyle.Render("Allowed"),
		target,
		"",
		m.filePicker.View(),
	}, "\n")
}

func (m Model) currentTemplateEntry() (TemplateEntry, bool) {
	if len(m.templates) == 0 {
		return TemplateEntry{}, false
	}
	if m.templateIndex < 0 || m.templateIndex >= len(m.templates) {
		return TemplateEntry{}, false
	}
	return m.templates[m.templateIndex], true
}

func (m Model) currentTemplateDecorationAssets() []domain.Asset {
	entry, ok := m.currentTemplateEntry()
	if !ok {
		return nil
	}
	return entry.DecorationAssets
}

func (m Model) additionalImageSlots() []domain.Slot {
	entry, ok := m.currentTemplateEntry()
	if !ok {
		return nil
	}
	return entry.AdditionalImageSlots()
}

func (m Model) syncSelectedTemplateFromList() Model {
	if len(m.templates) == 0 {
		return m
	}

	item, ok := m.templateList.SelectedItem().(templateItem)
	if !ok {
		return m
	}

	for idx, entry := range m.templates {
		if entry.ID == item.entry.ID {
			return m.selectTemplate(idx)
		}
	}

	return m
}

func (m Model) selectTemplate(idx int) Model {
	if idx < 0 || idx >= len(m.templates) {
		return m
	}

	entry := m.templates[idx]
	m.templateIndex = idx
	m.templateList.Select(idx)
	m.composition.Project.Template = entry.ID

	if len(entry.SupportedSizes) > 0 && !slices.Contains(entry.SupportedSizes, m.composition.Project.Page.Size) {
		m.composition.Project.Page.Size = entry.SupportedSizes[0]
	}
	if m.composition.Project.Page.Size == "" && len(entry.SupportedSizes) > 0 {
		m.composition.Project.Page.Size = entry.SupportedSizes[0]
	}
	if entry.DefaultOrientation != "" {
		m.composition.Project.Page.Orientation = entry.DefaultOrientation
	}

	m.composition.Project.Images = nil
	m.composition.DisableAllDecorations()
	m.decorationIndex = 0
	m.rebuildImageInputs(entry)
	m.syncSuggestedOutputPath(false)
	m.clearStatus(StepQuickStart)
	m.clearStatus(StepStyle)
	m.clearStatus(StepReview)
	return m
}

func (m *Model) rebuildImageInputs(entry TemplateEntry) {
	m.primaryImageInput.SetValue("")
	m.advancedInputs = map[string]textinput.Model{}

	if slot, ok := entry.PrimaryImageSlot(); ok {
		m.primaryImageInput.SetValue(m.composition.ImagePath(slot.ID))
	}

	for _, slot := range entry.AdditionalImageSlots() {
		input := newTextInput(slot.ID, "Paste an image path or browse")
		input.SetValue(m.composition.ImagePath(slot.ID))
		m.advancedInputs[slot.ID] = input
	}
}

func (m Model) cycleSize(delta int) Model {
	entry, ok := m.currentTemplateEntry()
	if !ok || len(entry.SupportedSizes) == 0 {
		return m
	}

	idx := slices.Index(entry.SupportedSizes, m.composition.Project.Page.Size)
	if idx == -1 {
		idx = 0
	}
	idx += delta
	for idx < 0 {
		idx += len(entry.SupportedSizes)
	}
	idx = idx % len(entry.SupportedSizes)
	m.composition.Project.Page.Size = entry.SupportedSizes[idx]
	return m
}

func (m Model) toggleOrientation() Model {
	if m.composition.Project.Page.Orientation == domain.OrientationLandscape {
		m.composition.Project.Page.Orientation = domain.OrientationPortrait
	} else {
		m.composition.Project.Page.Orientation = domain.OrientationLandscape
	}
	return m
}

func (m Model) toggleExportFormat(step Step) Model {
	if m.composition.Project.Export.Format == domain.ExportFormatPNG {
		m.composition.Project.Export.Format = domain.ExportFormatPDF
	} else {
		m.composition.Project.Export.Format = domain.ExportFormatPNG
	}
	m.syncSuggestedOutputPath(false)
	m.setStatus(step, fmt.Sprintf("Export format set to %s.", strings.ToUpper(string(m.composition.Project.Export.Format))), false)
	return m
}

func (m Model) applyBodyImport() Model {
	path := strings.TrimSpace(m.bodyImportInput.Value())
	body, err := importBodyFile(path)
	if err != nil {
		m.setStatus(StepQuickStart, err.Error(), true)
		return m
	}

	m.composition.BodyImportPath = path
	m.composition.Project.Content.Body = body
	m.bodyInput.SetValue(body)
	m.setStatus(StepQuickStart, fmt.Sprintf("Imported body text from %s.", path), false)
	return m
}

func importBodyFile(path string) (string, error) {
	if err := validateBodyImportFile(path); err != nil {
		return "", err
	}

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("could not read %s: %w", path, err)
	}

	return string(data), nil
}

func validateBodyImportFile(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("choose a .txt or .md file for the body")
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".txt" && ext != ".md" {
		return fmt.Errorf("body import supports only .txt and .md files")
	}
	info, err := os.Stat(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("body import file not found: %s", path)
	}
	if info.IsDir() {
		return fmt.Errorf("body import path must be a file")
	}
	return nil
}

func (m Model) applyPrimaryImageValue(path string, step Step) Model {
	entry, ok := m.currentTemplateEntry()
	if !ok {
		return m
	}
	slot, ok := entry.PrimaryImageSlot()
	if !ok {
		m.setStatus(step, "This template has no primary image slot.", true)
		return m
	}
	return m.applyImageValue(slot.ID, path, step)
}

func (m Model) applyImageValue(slotID, path string, step Step) Model {
	path = strings.TrimSpace(path)
	if path == "" {
		m.composition.RemoveImage(slotID)
		m.setStatus(step, fmt.Sprintf("Cleared image for %s.", slotID), false)
		return m
	}

	if err := validateImageFile(path); err != nil {
		m.setStatus(step, err.Error(), true)
		return m
	}

	m.composition.SetImage(slotID, path)
	m.setStatus(step, fmt.Sprintf("Image bound to %s.", slotID), false)
	return m
}

func validateImageFile(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("choose an image file")
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".webp":
	default:
		return fmt.Errorf("image field supports .png, .jpg, .jpeg, and .webp")
	}

	info, err := os.Stat(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("image file not found: %s", path)
	}
	if info.IsDir() {
		return fmt.Errorf("image path must be a file")
	}
	return nil
}

func (m Model) toggleDecorationsEnabled() Model {
	assets := m.currentTemplateDecorationAssets()
	if m.composition.Project.Options.Decorations {
		m.composition.DisableAllDecorations()
		m.setStatus(StepStyle, "Decorations disabled.", false)
		return m
	}
	m.composition.EnableAllDecorations(assets)
	m.setStatus(StepStyle, "Decorations enabled.", false)
	return m
}

func (m Model) toggleCurrentDecoration() Model {
	assets := m.currentTemplateDecorationAssets()
	if len(assets) == 0 {
		return m
	}
	if m.decorationIndex < 0 || m.decorationIndex >= len(assets) {
		m.decorationIndex = 0
	}
	asset := assets[m.decorationIndex]
	enabled := !m.composition.DecorationEnabled(asset.ID)
	m.composition.SetDecorationEnabled(asset.ID, enabled)
	if enabled {
		m.setStatus(StepStyle, fmt.Sprintf("Enabled decoration %s.", asset.ID), false)
	} else {
		m.setStatus(StepStyle, fmt.Sprintf("Disabled decoration %s.", asset.ID), false)
	}
	return m
}

func (m Model) exportComposition() Model {
	path := strings.TrimSpace(m.outputInput.Value())
	if path == "" {
		m.syncSuggestedOutputPath(true)
		path = strings.TrimSpace(m.outputInput.Value())
	}
	if path == "" {
		m.setStatus(StepReview, "Set an output path before exporting.", true)
		return m
	}

	m.composition.Project.Export.Out = path
	entry, ok := m.currentTemplateEntry()
	if !ok || entry.Template.ID == "" {
		m.setStatus(StepReview, "Select a template before exporting.", true)
		return m
	}

	projectPath := m.exportProjectPath()
	resolved, err := resolveTemplate(entry.Template, m.composition.Project)
	if err != nil {
		m.setStatus(StepReview, fmt.Sprintf("Template resolve failed: %v", err), true)
		return m
	}

	if err := saveProject(projectPath, m.composition.Project); err != nil {
		m.setStatus(StepReview, fmt.Sprintf("Project save failed: %v", err), true)
		return m
	}

	out, err := composeAndWrite(resolved, export.Options{
		Format:      m.composition.Project.Export.Format,
		Out:         path,
		Decorations: m.composition.Project.Options.Decorations,
	})
	if err != nil {
		m.setStatus(StepReview, fmt.Sprintf("Export failed: %v", err), true)
		return m
	}

	m.setStatus(StepReview, fmt.Sprintf("Export saved to %s. Project saved to %s.", out, projectPath), false)
	return m
}

func (m Model) exportProjectPath() string {
	output := strings.TrimSpace(m.composition.Project.Export.Out)
	if output == "" {
		output = m.suggestedOutputPath
	}
	if output == "" {
		return "exports/letterpress.project.yaml"
	}

	base := strings.TrimSuffix(output, filepath.Ext(output))
	if base == "" {
		base = "exports/letterpress"
	}
	return base + ".project.yaml"
}

func (m Model) openFilePicker(target fileTargetKind, slotID string) (tea.Model, tea.Cmd) {
	picker := filepicker.New()
	picker.CurrentDirectory = currentDirectoryForValue(m.currentPathValue(target, slotID))
	picker.AllowedTypes = allowedTypesForTarget(target)
	picker.ShowPermissions = false
	picker.ShowSize = false
	picker.FileAllowed = true
	picker.DirAllowed = false
	picker.SetHeight(m.modalHeight())
	picker.KeyMap.Back = key.NewBinding(key.WithKeys("backspace", "left"), key.WithHelp("backspace", "up"))
	picker.KeyMap.Down = key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "down"))
	picker.KeyMap.Up = key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "up"))

	m.filePicker = picker
	m.filePickerOpen = true
	m.fileTargetKind = target
	m.fileTargetSlotID = slotID
	return m, picker.Init()
}

func currentDirectoryForValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "."
		}
		return wd
	}

	cleaned := filepath.Clean(value)
	info, err := os.Stat(cleaned)
	if err == nil {
		if info.IsDir() {
			return cleaned
		}
		return filepath.Dir(cleaned)
	}

	dir := filepath.Dir(cleaned)
	if dir == "." || dir == "" {
		wd, wdErr := os.Getwd()
		if wdErr != nil {
			return "."
		}
		return wd
	}

	if _, err := os.Stat(dir); err == nil {
		return dir
	}

	wd, wdErr := os.Getwd()
	if wdErr != nil {
		return "."
	}
	return wd
}

func allowedTypesForTarget(target fileTargetKind) []string {
	switch target {
	case fileTargetBodyImport:
		return []string{".txt", ".md"}
	case fileTargetPrimaryImage, fileTargetSlotImage:
		return []string{".png", ".jpg", ".jpeg", ".webp"}
	default:
		return nil
	}
}

func (m Model) currentPathValue(target fileTargetKind, slotID string) string {
	switch target {
	case fileTargetBodyImport:
		return m.bodyImportInput.Value()
	case fileTargetPrimaryImage:
		return m.primaryImageInput.Value()
	case fileTargetSlotImage:
		if input, ok := m.advancedInputs[slotID]; ok {
			return input.Value()
		}
	}
	return ""
}

func (m *Model) applyFileSelection(path string) {
	cleaned := normalizeSelectedPath(path)
	switch m.fileTargetKind {
	case fileTargetBodyImport:
		m.bodyImportInput.SetValue(cleaned)
		updated := m.applyBodyImport()
		*m = updated
	case fileTargetPrimaryImage:
		m.primaryImageInput.SetValue(cleaned)
		updated := m.applyPrimaryImageValue(cleaned, StepQuickStart)
		*m = updated
	case fileTargetSlotImage:
		input := m.advancedInputs[m.fileTargetSlotID]
		input.SetValue(cleaned)
		m.advancedInputs[m.fileTargetSlotID] = input
		updated := m.applyImageValue(m.fileTargetSlotID, cleaned, StepStyle)
		*m = updated
	}
}

func normalizeSelectedPath(path string) string {
	cleaned := filepath.Clean(path)
	wd, err := os.Getwd()
	if err != nil {
		return cleaned
	}
	rel, err := filepath.Rel(wd, cleaned)
	if err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return rel
	}
	return cleaned
}

func (m *Model) updateOutputPathFromInput() {
	value := strings.TrimSpace(m.outputInput.Value())
	if value == "" {
		m.composition.Project.Export.Out = ""
		m.customOutputPath = false
		return
	}

	m.composition.Project.Export.Out = value
	m.customOutputPath = value != m.suggestedOutputPath
}

func (m Model) suggestedOutput() string {
	entry, ok := m.currentTemplateEntry()
	name := "letterpress"
	if ok && entry.ID != "" {
		name = entry.ID
	}

	format := m.composition.Project.Export.Format
	if format == "" {
		format = domain.ExportFormatPDF
	}

	return filepath.Join("exports", name+"."+string(format))
}

func (m *Model) syncSuggestedOutputPath(force bool) {
	oldSuggested := m.suggestedOutputPath
	newSuggested := m.suggestedOutput()
	current := strings.TrimSpace(m.outputInput.Value())
	shouldApply := force || !m.customOutputPath || current == "" || current == oldSuggested

	m.suggestedOutputPath = newSuggested
	if shouldApply {
		m.outputInput.SetValue(newSuggested)
		m.composition.Project.Export.Out = newSuggested
		m.customOutputPath = false
	}
}

func (m *Model) setStatus(step Step, text string, isError bool) {
	if m.statuses == nil {
		m.statuses = map[Step]stepStatus{}
	}
	m.statuses[step] = stepStatus{Text: text, IsError: isError}
}

func (m *Model) clearStatus(step Step) {
	delete(m.statuses, step)
}

func (m Model) status(step Step) stepStatus {
	return m.statuses[step]
}

func (m Model) bodyImportSummary() string {
	applied := strings.TrimSpace(m.composition.BodyImportPath)
	pending := strings.TrimSpace(m.bodyImportInput.Value())

	switch {
	case pending != "" && pending != applied:
		return "Pending: " + summarizeText(pending)
	case applied != "":
		return summarizeText(applied)
	default:
		return "Manual editing"
	}
}

func (m Model) decorationsToggleLabel() string {
	if m.composition.Project.Options.Decorations {
		return "Decorations On"
	}
	return "Decorations Off"
}

func joinPanels(left, right string, width int) string {
	if right == "" {
		return left
	}
	if width < 120 {
		return lipgloss.JoinVertical(lipgloss.Left, left, "", right)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
}

func (m Model) fullPanelWidth() int {
	if m.width <= 0 {
		return 104
	}
	return max(64, m.width-8)
}

func (m Model) mainPanelWidth() int {
	if m.width <= 0 {
		return 72
	}
	if m.width < 120 {
		return max(52, m.width-8)
	}
	return int(float64(m.width) * 0.68)
}

func (m Model) summaryWidth() int {
	if m.width <= 0 {
		return 30
	}
	if m.width < 120 {
		return max(30, m.width-8)
	}
	return max(30, m.width-m.mainPanelWidth()-10)
}

func (m Model) modalHeight() int {
	return min(16, max(8, m.height-16))
}

func summarizeText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "(empty)"
	}
	if len(value) <= 56 {
		return value
	}
	return value[:53] + "..."
}

func styleSlotFocus(slotID string) string {
	return "style-slot:" + slotID
}

func styleSlotBrowseFocus(slotID string) string {
	return "style-slot-browse:" + slotID
}

func styleSlotIDFromFocus(focus string) (string, bool) {
	if !strings.HasPrefix(focus, "style-slot:") {
		return "", false
	}
	return strings.TrimPrefix(focus, "style-slot:"), true
}

func styleSlotBrowseIDFromFocus(focus string) (string, bool) {
	if !strings.HasPrefix(focus, "style-slot-browse:") {
		return "", false
	}
	return strings.TrimPrefix(focus, "style-slot-browse:"), true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
