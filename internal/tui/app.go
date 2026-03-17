package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaeyoung0509/letterpress/internal/domain"
)

type Step string

const (
	StepTemplate    Step = "Template Selection"
	StepSize        Step = "Paper Size & Orientation"
	StepContent     Step = "Content Composition"
	StepImages      Step = "Image Assignment"
	StepDecorations Step = "Decoration Selection"
	StepReview      Step = "Review & Export"
)

var stepOrder = []Step{
	StepTemplate,
	StepSize,
	StepContent,
	StepImages,
	StepDecorations,
	StepReview,
}

type RouteState struct {
	Title       string
	Description string
	Placeholder string
}

type State struct {
	Current Step
	Routes  map[Step]RouteState
}

type ContentField int

const (
	FieldTitle ContentField = iota
	FieldBody
	FieldSignature
)

func newState() State {
	return State{
		Current: StepTemplate,
		Routes: map[Step]RouteState{
			StepTemplate: {
				Title:       "Template Selection",
				Description: "Placeholder route for curated templates and layouts.",
				Placeholder: "Use j/k to pick templates and Enter to continue.",
			},
			StepSize: {
				Title:       "Paper Size & Orientation",
				Description: "Frame the composition for A3–A6 in portrait or landscape.",
				Placeholder: "Use s to cycle sizes and o to toggle orientation.",
			},
			StepContent: {
				Title:       "Content Composition",
				Description: "Compose title, body, signature, and decorative slots.",
				Placeholder: "Type text, use Tab to switch fields.",
			},
			StepImages: {
				Title:       "Image Assignment",
				Description: "Bind local photos to the template's image slots.",
				Placeholder: "Use j/k to switch slots, type path, and = to assign.",
			},
			StepDecorations: {
				Title:       "Decoration Selection",
				Description: "Enable or disable decorative assets.",
				Placeholder: "Use n/p to pick assets and t to toggle each.",
			},
			StepReview: {
				Title:       "Review & Export",
				Description: "Finalize layout, toggle decorations, and export.",
				Placeholder: "Future work: show export targets (PDF/PNG) and notes.",
			},
		},
	}
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
	Forward string
	Back    string
	Quit    string
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Forward: "enter / right / down",
		Back:    "backspace / left / up",
		Quit:    "q / ctrl+c",
	}
}

type Model struct {
	state           State
	composition     CompositionState
	keyMap          KeyMap
	width           int
	height          int
	templates       []TemplateEntry
	templateIndex   int
	contentField    ContentField
	imageSlotIndex  int
	imageInput      string
	decorationIndex int
}

func NewModel() Model {
	model := Model{
		state:        newState(),
		composition:  newCompositionState(),
		keyMap:       DefaultKeyMap(),
		templates:    loadTemplateEntries(),
		contentField: FieldTitle,
	}

	if len(model.templates) > 0 {
		model = model.selectTemplate(0)
	}

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
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "enter", "right", "down", " ":
		m.state = m.state.withNext()
		return m, nil
	case "esc", "left", "up":
		m.state = m.state.withPrev()
		return m, nil
	case "backspace":
		switch m.state.Current {
		case StepContent:
			m = m.deleteContentRune()
		case StepImages:
			m.imageInput = trimLastRune(m.imageInput)
		default:
			m.state = m.state.withPrev()
		}
		return m, nil
	}

	switch m.state.Current {
	case StepTemplate:
		m = m.handleTemplateKey(msg)
	case StepSize:
		m = m.handleSizeKey(msg)
	case StepContent:
		m = m.handleContentKey(msg)
	case StepImages:
		m = m.handleImageKey(msg)
	case StepDecorations:
		m = m.handleDecorationKey(msg)
	}

	return m, nil
}

func (m Model) handleTemplateKey(msg tea.KeyMsg) Model {
	switch strings.ToLower(msg.String()) {
	case "j":
		return m.cycleTemplate(1)
	case "k":
		return m.cycleTemplate(-1)
	}

	return m
}

func (m Model) handleSizeKey(msg tea.KeyMsg) Model {
	switch strings.ToLower(msg.String()) {
	case "s":
		return m.cycleSize(1)
	case "o":
		return m.toggleOrientation()
	}

	return m
}

func (m Model) handleContentKey(msg tea.KeyMsg) Model {
	switch msg.Type {
	case tea.KeyTab:
		return m.cycleContentField()
	case tea.KeyRunes:
		return m.appendContentRunes(msg.Runes)
	}
	return m
}

func (m Model) handleImageKey(msg tea.KeyMsg) Model {
	switch strings.ToLower(msg.String()) {
	case "j":
		return m.cycleImageSlot(1)
	case "k":
		return m.cycleImageSlot(-1)
	case "=":
		return m.assignCurrentImage()
	}

	switch msg.Type {
	case tea.KeyRunes:
		m.imageInput += string(msg.Runes)
	}

	return m
}

func (m Model) handleDecorationKey(msg tea.KeyMsg) Model {
	switch strings.ToLower(msg.String()) {
	case "n":
		return m.cycleDecoration(1)
	case "p":
		return m.cycleDecoration(-1)
	case "t":
		return m.toggleCurrentDecoration()
	}

	return m
}

func (m Model) View() string {
	lines := []string{
		"letterpress",
		"Bubble Tea composition shell",
		"",
		m.renderSteps(),
		"",
		m.renderRoute(),
		"",
		m.renderStepContent(),
		"",
		m.renderCompositionSummary(),
		"",
		fmt.Sprintf("Viewport: %dx%d", m.width, m.height),
		"",
		m.renderKeyLegend(),
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderSteps() string {
	var steps []string
	steps = append(steps, "Steps:")
	for _, step := range stepOrder {
		prefix := "   "
		if step == m.state.Current {
			prefix = "→ "
		}
		steps = append(steps, fmt.Sprintf("%s%s", prefix, step))
	}

	return strings.Join(steps, "\n")
}

func (m Model) renderRoute() string {
	route, ok := m.state.Routes[m.state.Current]
	if !ok {
		return "Route placeholder unavailable."
	}

	return fmt.Sprintf("%s\n%s\n\n%s", route.Title, route.Description, route.Placeholder)
}

func (m Model) renderCompositionSummary() string {
	return fmt.Sprintf("Composition in progress (%s)", m.composition.Summary())
}

func (m Model) renderKeyLegend() string {
	return fmt.Sprintf("[Forward: %s]  [Back: %s]  [Quit: %s]", m.keyMap.Forward, m.keyMap.Back, m.keyMap.Quit)
}

func (m Model) renderStepContent() string {
	switch m.state.Current {
	case StepTemplate:
		return m.renderTemplatePicker()
	case StepSize:
		return m.renderSizeSelector()
	case StepContent:
		return m.renderContentEditor()
	case StepImages:
		return m.renderImageAssignment()
	case StepDecorations:
		return m.renderDecorationSelection()
	default:
		return ""
	}
}

func (m Model) renderTemplatePicker() string {
	if len(m.templates) == 0 {
		return "Template catalog unavailable."
	}

	var builder strings.Builder
	builder.WriteString("Available templates:")
	for idx, entry := range m.templates {
		prefix := "   "
		if idx == m.templateIndex {
			prefix = "→ "
		}
		builder.WriteString("\n")
		builder.WriteString(fmt.Sprintf("%s%s", prefix, entry.Label()))
		builder.WriteString("\n")
		builder.WriteString(fmt.Sprintf("      sizes: %s", m.formatSizes(entry.SupportedSizes)))
	}

	builder.WriteString("\n\nUse j/k to cycle templates, Enter to continue.")
	return builder.String()
}

func (m Model) renderSizeSelector() string {
	entry, ok := m.currentTemplateEntry()
	if !ok {
		return "Select a template to configure page size."
	}

	var builder strings.Builder
	builder.WriteString("Supported sizes:")
	for _, size := range entry.SupportedSizes {
		prefix := "   "
		if size == m.composition.Project.Page.Size {
			prefix = "→ "
		}
		builder.WriteString("\n")
		builder.WriteString(fmt.Sprintf("%s%s", prefix, size))
	}

	builder.WriteString("\n\nOrientation: ")
	builder.WriteString(string(m.composition.Project.Page.Orientation))
	builder.WriteString("\n\nPress s to cycle sizes, o to toggle orientation.")
	return builder.String()
}

func (m Model) renderContentEditor() string {
	fields := []struct {
		field ContentField
		label string
	}{
		{FieldTitle, "Title"},
		{FieldBody, "Body"},
		{FieldSignature, "Signature"},
	}

	var builder strings.Builder
	builder.WriteString("Content fields:")
	for _, entry := range fields {
		prefix := "   "
		if entry.field == m.contentField {
			prefix = "→ "
		}
		value := m.fieldValue(entry.field)
		if value == "" {
			value = "(empty)"
		}
		builder.WriteString("\n")
		builder.WriteString(fmt.Sprintf("%s%s: %s", prefix, entry.label, value))
	}

	builder.WriteString("\n\nType to edit text, Tab to switch fields.")
	return builder.String()
}

func (m Model) renderImageAssignment() string {
	slots := m.currentTemplateImageSlots()
	if len(slots) == 0 {
		return "No image slots defined for the selected template."
	}

	var builder strings.Builder
	builder.WriteString("Image slots:")
	for idx, slot := range slots {
		prefix := "   "
		if idx == m.imageSlotIndex {
			prefix = "→ "
		}
		path := m.imagePathForSlot(slot.ID)
		if path == "" {
			path = "(unassigned)"
		}
		builder.WriteString("\n")
		builder.WriteString(fmt.Sprintf("%s%s: %s", prefix, slot.ID, path))
	}

	builder.WriteString("\n\nInput path: ")
	if m.imageInput == "" {
		builder.WriteString("(empty)")
	} else {
		builder.WriteString(m.imageInput)
	}

	builder.WriteString("\n\nType path, use j/k to select slot, = to assign.")
	return builder.String()
}

func (m Model) renderDecorationSelection() string {
	assets := m.currentTemplateDecorationAssets()
	if len(assets) == 0 {
		return "No decoration assets defined for this template."
	}

	var builder strings.Builder
	builder.WriteString("Decoration assets:")
	for idx, asset := range assets {
		prefix := "   "
		if idx == m.decorationIndex {
			prefix = "→ "
		}
		status := "off"
		if m.composition.DecorationSelections[asset.ID] {
			status = "on"
		}
		builder.WriteString("\n")
		builder.WriteString(fmt.Sprintf("%s%s [%s]", prefix, asset.ID, status))
	}

	builder.WriteString("\n\nUse n/p to cycle, t to toggle on/off.")
	return builder.String()
}

func (m Model) currentImageSlot() (domain.Slot, bool) {
	slots := m.currentTemplateImageSlots()
	if len(slots) == 0 {
		return domain.Slot{}, false
	}

	idx := m.imageSlotIndex
	if idx < 0 || idx >= len(slots) {
		idx = 0
	}

	return slots[idx], true
}

func (m Model) cycleImageSlot(delta int) Model {
	slots := m.currentTemplateImageSlots()
	if len(slots) == 0 {
		return m
	}

	next := m.imageSlotIndex + delta
	for next < 0 {
		next += len(slots)
	}
	m.imageSlotIndex = next % len(slots)
	m.imageInput = ""
	return m
}

func (m Model) assignCurrentImage() Model {
	slot, ok := m.currentImageSlot()
	if !ok {
		return m
	}

	path := strings.TrimSpace(m.imageInput)
	if path == "" {
		return m
	}

	m = m.setImageForSlot(slot.ID, path)
	m.imageInput = ""
	return m
}

func (m Model) setImageForSlot(slotID, path string) Model {
	for idx, binding := range m.composition.Project.Images {
		if binding.Slot == slotID {
			m.composition.Project.Images[idx].Path = path
			return m
		}
	}

	m.composition.Project.Images = append(m.composition.Project.Images, domain.ImageBinding{
		Slot: slotID,
		Path: path,
	})

	return m
}

func (m Model) imagePathForSlot(slotID string) string {
	for _, binding := range m.composition.Project.Images {
		if binding.Slot == slotID {
			return binding.Path
		}
	}

	return ""
}

func (m Model) currentDecorationAsset() (domain.Asset, bool) {
	assets := m.currentTemplateDecorationAssets()
	if len(assets) == 0 {
		return domain.Asset{}, false
	}

	idx := m.decorationIndex
	if idx < 0 || idx >= len(assets) {
		idx = 0
	}

	return assets[idx], true
}

func (m Model) cycleDecoration(delta int) Model {
	assets := m.currentTemplateDecorationAssets()
	if len(assets) == 0 {
		return m
	}

	next := m.decorationIndex + delta
	for next < 0 {
		next += len(assets)
	}
	m.decorationIndex = next % len(assets)
	return m
}

func (m Model) toggleCurrentDecoration() Model {
	asset, ok := m.currentDecorationAsset()
	if !ok {
		return m
	}

	if m.composition.DecorationSelections == nil {
		m.composition.DecorationSelections = map[string]bool{}
	}

	current := m.composition.DecorationSelections[asset.ID]
	if current {
		delete(m.composition.DecorationSelections, asset.ID)
	} else {
		m.composition.DecorationSelections[asset.ID] = true
	}

	return m
}

func trimLastRune(value string) string {
	if value == "" {
		return value
	}

	_, size := utf8.DecodeLastRuneInString(value)
	return value[:len(value)-size]
}

func (m Model) formatSizes(sizes []domain.PageSize) string {
	if len(sizes) == 0 {
		return "none"
	}

	parts := make([]string, len(sizes))
	for i, size := range sizes {
		parts[i] = string(size)
	}

	return strings.Join(parts, ", ")
}

func (m Model) currentTemplateEntry() (TemplateEntry, bool) {
	if len(m.templates) == 0 {
		return TemplateEntry{}, false
	}

	idx := m.templateIndex
	if idx < 0 || idx >= len(m.templates) {
		idx = 0
	}

	return m.templates[idx], true
}

func (m Model) selectTemplate(idx int) Model {
	if len(m.templates) == 0 {
		return m
	}

	if idx < 0 {
		idx = 0
	} else if idx >= len(m.templates) {
		idx = len(m.templates) - 1
	}

	entry := m.templates[idx]
	m.templateIndex = idx
	m.composition.Project.Template = entry.ID
	if len(entry.SupportedSizes) > 0 {
		m.composition.Project.Page.Size = entry.SupportedSizes[0]
	}
	m.composition.Project.Page.Orientation = entry.DefaultOrientation
	m.imageSlotIndex = 0
	m.imageInput = ""
	m.decorationIndex = 0
	m.composition.Project.Images = nil
	m.composition.DecorationSelections = map[string]bool{}

	return m
}

func (m Model) currentTemplateImageSlots() []domain.Slot {
	entry, ok := m.currentTemplateEntry()
	if !ok {
		return nil
	}

	return entry.ImageSlots
}

func (m Model) currentTemplateDecorationAssets() []domain.Asset {
	entry, ok := m.currentTemplateEntry()
	if !ok {
		return nil
	}

	return entry.DecorationAssets
}

func (m Model) cycleTemplate(delta int) Model {
	if len(m.templates) == 0 {
		return m
	}

	next := m.templateIndex + delta
	for next < 0 {
		next += len(m.templates)
	}
	next = next % len(m.templates)

	return m.selectTemplate(next)
}

func (m Model) cycleSize(delta int) Model {
	entry, ok := m.currentTemplateEntry()
	if !ok {
		return m
	}

	sizes := entry.SupportedSizes
	if len(sizes) == 0 {
		return m
	}

	current := m.composition.Project.Page.Size
	idx := 0
	for i, size := range sizes {
		if size == current {
			idx = i
			break
		}
	}

	next := idx + delta
	for next < 0 {
		next += len(sizes)
	}
	next = next % len(sizes)

	m.composition.Project.Page.Size = sizes[next]
	return m
}

func (m Model) toggleOrientation() Model {
	current := m.composition.Project.Page.Orientation
	if current == domain.OrientationLandscape {
		m.composition.Project.Page.Orientation = domain.OrientationPortrait
	} else {
		m.composition.Project.Page.Orientation = domain.OrientationLandscape
	}

	return m
}

func (m Model) appendContentRunes(runes []rune) Model {
	if len(runes) == 0 {
		return m
	}

	value := m.fieldValue(m.contentField)
	value += string(runes)
	return m.setFieldValue(m.contentField, value)
}

func (m Model) deleteContentRune() Model {
	value := m.fieldValue(m.contentField)
	if value == "" {
		return m
	}

	_, size := utf8.DecodeLastRuneInString(value)
	value = value[:len(value)-size]
	return m.setFieldValue(m.contentField, value)
}

func (m Model) cycleContentField() Model {
	m.contentField++
	if m.contentField > FieldSignature {
		m.contentField = FieldTitle
	}

	return m
}

func (m Model) fieldValue(field ContentField) string {
	switch field {
	case FieldTitle:
		return m.composition.Project.Content.Title
	case FieldBody:
		return m.composition.Project.Content.Body
	case FieldSignature:
		return m.composition.Project.Content.Signature
	default:
		return ""
	}
}

func (m Model) setFieldValue(field ContentField, value string) Model {
	switch field {
	case FieldTitle:
		m.composition.Project.Content.Title = value
	case FieldBody:
		m.composition.Project.Content.Body = value
	case FieldSignature:
		m.composition.Project.Content.Signature = value
	}

	return m
}
