package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	asciipkg "github.com/jaeyoung0509/letterpress/internal/ascii"
	"github.com/jaeyoung0509/letterpress/internal/domain"
	exportpkg "github.com/jaeyoung0509/letterpress/internal/export"
	renderpkg "github.com/jaeyoung0509/letterpress/internal/render"
)

var (
	composeASCII         = renderpkg.ComposeASCII
	composeAndWriteASCII = exportpkg.ComposeAndWriteASCII
)

type KeyMap struct {
	Run         key.Binding
	HistoryPrev key.Binding
	HistoryNext key.Binding
	Clear       key.Binding
	Quit        key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Run:         key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "run command")),
		HistoryPrev: key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "history prev")),
		HistoryNext: key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "history next")),
		Clear:       key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear input")),
		Quit:        key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Run, k.HistoryPrev, k.HistoryNext, k.Clear, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Run, k.HistoryPrev, k.HistoryNext, k.Clear, k.Quit}}
}

type statusMessage struct {
	Text    string
	IsError bool
}

type Draft struct {
	Mode         domain.RenderMode
	Page         domain.ProjectPage
	ASCII        domain.ASCIIOptions
	ImagePath    string
	Text         string
	TextFile     string
	ExportFormat exportpkg.ASCIIFormat
	ExportOut    string
}

func newDraft() Draft {
	return Draft{
		Mode: domain.RenderModeASCII,
		Page: domain.ProjectPage{
			Size:        domain.PageSizeA4,
			Orientation: domain.OrientationPortrait,
		},
		ASCII: domain.ASCIIOptions{
			Density: 80,
		},
		ExportFormat: exportpkg.ASCIIFormatPDF,
		ExportOut:    suggestedExportPath(exportpkg.ASCIIFormatPDF),
	}
}

func (d Draft) EffectiveASCIIOptions() domain.ASCIIOptions {
	opts := d.ASCII
	if strings.TrimSpace(opts.Charset) == "" {
		if derived := normalizeGlyphSource(d.Text); derived != "" {
			opts.Charset = derived
		}
	}
	return opts
}

func (d Draft) GlyphSourceLabel() string {
	if strings.TrimSpace(d.ASCII.Charset) != "" {
		return "explicit charset"
	}
	if strings.TrimSpace(d.Text) != "" {
		return "derived from lettering"
	}
	return "default ascii ramp"
}

type slashCommand struct {
	Name string
	Args []string
}

type Model struct {
	width  int
	height int

	keyMap KeyMap
	help   help.Model

	commandInput textinput.Model
	status       statusMessage

	draft Draft

	preview    asciipkg.Art
	previewErr string

	history      []string
	historyIndex int
	historyDraft string
}

func NewModel() Model {
	commandInput := textinput.New()
	commandInput.Prompt = ""
	commandInput.Placeholder = `/image ./test.png`
	commandInput.Width = 72
	commandInput.Focus()

	model := Model{
		width:        120,
		height:       38,
		keyMap:       DefaultKeyMap(),
		help:         help.New(),
		commandInput: commandInput,
		draft:        newDraft(),
		historyIndex: -1,
	}
	model.refreshPreview()
	return model
}

func Run() error {
	program := tea.NewProgram(NewModel(), tea.WithAltScreen())
	_, err := program.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.commandInput.Width = max(28, m.width-12)
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keyMap.Run):
			m = m.executeCurrentCommand()
			return m, nil
		case key.Matches(msg, m.keyMap.HistoryPrev):
			m.historyPrev()
			return m, nil
		case key.Matches(msg, m.keyMap.HistoryNext):
			m.historyNext()
			return m, nil
		case key.Matches(msg, m.keyMap.Clear):
			m.commandInput.SetValue("")
			m.setStatus("Command cleared.", false)
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.commandInput, cmd = m.commandInput.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	header := strings.Join([]string{
		headerStyle.Render("letterpress"),
		subtitleStyle.Render("ASCII mode with slash-command configuration."),
	}, "\n")

	mainWidth := max(52, int(float64(max(80, m.width))*0.62))
	sideWidth := max(30, max(80, m.width)-mainWidth-6)

	left := panelStyle.Width(mainWidth).Render(m.renderPreviewPanel(mainWidth - 6))
	right := summaryPanelStyle.Width(sideWidth).Render(m.renderSummaryPanel(sideWidth - 6))
	body := joinPanels(left, right, max(80, m.width)-4)

	commandPanel := panelStyle.Width(max(60, m.width-4)).Render(m.renderCommandPanel())
	footer := mutedStyle.Render(m.help.View(m.keyMap))

	return appFrameStyle.Width(max(72, m.width)).Render(strings.Join([]string{
		header,
		"",
		body,
		"",
		commandPanel,
		"",
		footer,
	}, "\n"))
}

func (m *Model) executeCurrentCommand() Model {
	commandText := strings.TrimSpace(m.commandInput.Value())
	if commandText == "" {
		m.setStatus("Enter a slash command first.", true)
		return *m
	}

	command, err := parseSlashCommand(commandText)
	if err != nil {
		m.setStatus(err.Error(), true)
		return *m
	}

	m.pushHistory(commandText)
	m.commandInput.SetValue("")

	if err := m.applyCommand(command); err != nil {
		m.setStatus(err.Error(), true)
		return *m
	}

	return *m
}

func parseSlashCommand(input string) (slashCommand, error) {
	tokens, err := tokenizeCommand(strings.TrimSpace(input))
	if err != nil {
		return slashCommand{}, err
	}
	if len(tokens) == 0 {
		return slashCommand{}, fmt.Errorf("enter a slash command")
	}
	if !strings.HasPrefix(tokens[0], "/") {
		return slashCommand{}, fmt.Errorf("commands must start with /")
	}

	name := strings.TrimPrefix(strings.ToLower(tokens[0]), "/")
	if name == "" {
		return slashCommand{}, fmt.Errorf("command name is required")
	}

	return slashCommand{Name: name, Args: tokens[1:]}, nil
}

func tokenizeCommand(input string) ([]string, error) {
	var (
		tokens  []string
		current strings.Builder
		quote   rune
		escaped bool
	)

	flush := func() {
		if current.Len() == 0 {
			return
		}
		tokens = append(tokens, current.String())
		current.Reset()
	}

	for _, r := range input {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case quote != 0:
			if r == quote {
				quote = 0
				continue
			}
			current.WriteRune(r)
		case r == '"' || r == '\'':
			quote = r
		case unicode.IsSpace(r):
			flush()
		default:
			current.WriteRune(r)
		}
	}

	if escaped {
		return nil, fmt.Errorf("command cannot end with a trailing escape")
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quoted argument")
	}

	flush()
	return tokens, nil
}

func (m *Model) applyCommand(command slashCommand) error {
	switch command.Name {
	case "help":
		m.setStatus("Commands: /image /text-file /text /mode /size /orientation /charset /density /threshold /invert /export", false)
		return nil
	case "image":
		if len(command.Args) != 1 {
			return fmt.Errorf("usage: /image <path>")
		}
		if err := validateImageFile(command.Args[0]); err != nil {
			return err
		}
		m.draft.ImagePath = command.Args[0]
		m.refreshPreview()
		m.setStatus(fmt.Sprintf("Image set to %s.", command.Args[0]), false)
		return nil
	case "text-file":
		if len(command.Args) != 1 {
			return fmt.Errorf("usage: /text-file <path>")
		}
		body, err := importTextFile(command.Args[0])
		if err != nil {
			return err
		}
		m.draft.TextFile = command.Args[0]
		m.draft.Text = body
		m.refreshPreview()
		m.setStatus(fmt.Sprintf("Loaded lettering text from %s.", command.Args[0]), false)
		return nil
	case "text":
		if len(command.Args) == 0 {
			return fmt.Errorf("usage: /text <lettering>")
		}
		m.draft.Text = strings.Join(command.Args, " ")
		m.draft.TextFile = ""
		m.refreshPreview()
		m.setStatus("Lettering text updated.", false)
		return nil
	case "mode":
		if len(command.Args) != 1 || strings.ToLower(command.Args[0]) != string(domain.RenderModeASCII) {
			return fmt.Errorf("usage: /mode ascii")
		}
		m.draft.Mode = domain.RenderModeASCII
		m.refreshPreview()
		m.setStatus("Render mode set to ascii.", false)
		return nil
	case "size":
		if len(command.Args) != 1 {
			return fmt.Errorf("usage: /size <A3|A4|A5|A6>")
		}
		size := domain.PageSize(strings.ToUpper(command.Args[0]))
		switch size {
		case domain.PageSizeA3, domain.PageSizeA4, domain.PageSizeA5, domain.PageSizeA6:
			m.draft.Page.Size = size
			m.refreshPreview()
			m.setStatus(fmt.Sprintf("Page size set to %s.", size), false)
			return nil
		default:
			return fmt.Errorf("page size must be A3, A4, A5, or A6")
		}
	case "orientation":
		if len(command.Args) != 1 {
			return fmt.Errorf("usage: /orientation <portrait|landscape>")
		}
		orientation := domain.Orientation(strings.ToLower(command.Args[0]))
		switch orientation {
		case domain.OrientationPortrait, domain.OrientationLandscape:
			m.draft.Page.Orientation = orientation
			m.refreshPreview()
			m.setStatus(fmt.Sprintf("Orientation set to %s.", orientation), false)
			return nil
		default:
			return fmt.Errorf("orientation must be portrait or landscape")
		}
	case "charset":
		if len(command.Args) == 0 {
			return fmt.Errorf("usage: /charset <glyphs>")
		}
		m.draft.ASCII.Charset = strings.Join(command.Args, " ")
		m.refreshPreview()
		m.setStatus("Explicit charset updated.", false)
		return nil
	case "density":
		if len(command.Args) != 1 {
			return fmt.Errorf("usage: /density <columns>")
		}
		density, err := strconv.Atoi(command.Args[0])
		if err != nil || density <= 0 {
			return fmt.Errorf("density must be a positive integer")
		}
		m.draft.ASCII.Density = density
		m.refreshPreview()
		m.setStatus(fmt.Sprintf("Density set to %d.", density), false)
		return nil
	case "threshold":
		if len(command.Args) != 1 {
			return fmt.Errorf("usage: /threshold <0..1>")
		}
		threshold, err := strconv.ParseFloat(command.Args[0], 64)
		if err != nil || threshold < 0 || threshold > 1 {
			return fmt.Errorf("threshold must be between 0 and 1")
		}
		m.draft.ASCII.Threshold = threshold
		m.refreshPreview()
		m.setStatus(fmt.Sprintf("Threshold set to %.2f.", threshold), false)
		return nil
	case "invert":
		if len(command.Args) != 1 {
			return fmt.Errorf("usage: /invert <on|off>")
		}
		switch strings.ToLower(command.Args[0]) {
		case "on", "true", "yes":
			m.draft.ASCII.Invert = true
		case "off", "false", "no":
			m.draft.ASCII.Invert = false
		default:
			return fmt.Errorf("invert must be on or off")
		}
		m.refreshPreview()
		m.setStatus(fmt.Sprintf("Invert set to %t.", m.draft.ASCII.Invert), false)
		return nil
	case "export":
		return m.handleExportCommand(command.Args)
	default:
		return fmt.Errorf("unknown command /%s", command.Name)
	}
}

func (m *Model) handleExportCommand(args []string) error {
	if len(args) == 0 || len(args) > 2 {
		return fmt.Errorf("usage: /export <txt|pdf> [path]")
	}
	if strings.TrimSpace(m.draft.ImagePath) == "" {
		return fmt.Errorf("set an image first with /image <path>")
	}

	format := exportpkg.ASCIIFormat(strings.ToLower(args[0]))
	if format != exportpkg.ASCIIFormatTXT && format != exportpkg.ASCIIFormatPDF {
		return fmt.Errorf("export format must be txt or pdf")
	}

	target := m.draft.ExportOut
	if len(args) == 2 {
		target = args[1]
	}
	if strings.TrimSpace(target) == "" {
		target = suggestedExportPath(format)
	}

	out, err := composeAndWriteASCII(m.draft.Page, m.draft.ImagePath, m.draft.EffectiveASCIIOptions(), exportpkg.ASCIIExportOptions{
		Format: format,
		Out:    target,
	})
	if err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	m.draft.ExportFormat = format
	m.draft.ExportOut = target
	m.setStatus(fmt.Sprintf("Export saved to %s.", out), false)
	return nil
}

func (m *Model) refreshPreview() {
	if strings.TrimSpace(m.draft.ImagePath) == "" {
		m.preview = asciipkg.Art{}
		m.previewErr = "Use /image <path> to generate a preview."
		return
	}

	composition, err := composeASCII(m.draft.Page, m.draft.ImagePath, m.draft.EffectiveASCIIOptions())
	if err != nil {
		m.preview = asciipkg.Art{}
		m.previewErr = err.Error()
		return
	}

	m.preview = composition.Art
	m.previewErr = ""
}

func (m *Model) setStatus(text string, isError bool) {
	m.status = statusMessage{Text: text, IsError: isError}
}

func (m *Model) pushHistory(command string) {
	if len(m.history) == 0 || m.history[len(m.history)-1] != command {
		m.history = append(m.history, command)
	}
	m.historyIndex = len(m.history)
	m.historyDraft = ""
}

func (m *Model) historyPrev() {
	if len(m.history) == 0 {
		return
	}
	if m.historyIndex == -1 {
		m.historyIndex = len(m.history)
	}
	if m.historyIndex == len(m.history) {
		m.historyDraft = m.commandInput.Value()
	}
	if m.historyIndex > 0 {
		m.historyIndex--
	}
	m.commandInput.SetValue(m.history[m.historyIndex])
	m.commandInput.CursorEnd()
}

func (m *Model) historyNext() {
	if len(m.history) == 0 || m.historyIndex == -1 {
		return
	}
	if m.historyIndex < len(m.history)-1 {
		m.historyIndex++
		m.commandInput.SetValue(m.history[m.historyIndex])
		m.commandInput.CursorEnd()
		return
	}
	m.historyIndex = len(m.history)
	m.commandInput.SetValue(m.historyDraft)
	m.commandInput.CursorEnd()
}

func (m Model) renderPreviewPanel(width int) string {
	lines := []string{
		sectionTitleStyle.Render("ASCII Preview"),
		mutedStyle.Render("Preview refreshes after each valid command."),
	}

	if m.previewErr != "" {
		lines = append(lines, "", errorStyle.Render(m.previewErr))
	} else {
		lines = append(lines, "", previewFrameStyle.Width(max(24, width-2)).Render(m.renderPreviewArt(width-8)))
	}

	if strings.TrimSpace(m.draft.Text) != "" {
		lines = append(lines, "",
			labelStyle.Render("Lettering seed"),
			mutedStyle.Render(summarizeText(m.draft.Text)),
		)
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderPreviewArt(width int) string {
	if len(m.preview.Lines) == 0 {
		return mutedStyle.Render("(no preview)")
	}

	maxLines := max(6, min(18, m.height-18))
	lines := make([]string, 0, maxLines)
	for idx, line := range m.preview.Lines {
		if idx == maxLines {
			lines = append(lines, mutedStyle.Render("..."))
			break
		}
		lines = append(lines, truncateForWidth(line, width))
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderSummaryPanel(width int) string {
	lines := []string{
		sectionTitleStyle.Render("Current State"),
		"",
		labelStyle.Render("Mode"),
		string(m.draft.Mode),
		labelStyle.Render("Page"),
		fmt.Sprintf("%s %s", m.draft.Page.Size, m.draft.Page.Orientation),
		labelStyle.Render("Image"),
		summarizeText(m.draft.ImagePath),
		labelStyle.Render("Text file"),
		summarizeText(m.draft.TextFile),
		labelStyle.Render("Glyph source"),
		m.draft.GlyphSourceLabel(),
		labelStyle.Render("Density"),
		fmt.Sprintf("%d columns", m.effectiveDensity()),
		labelStyle.Render("Threshold"),
		formatThreshold(m.draft.ASCII.Threshold),
		labelStyle.Render("Invert"),
		fmt.Sprintf("%t", m.draft.ASCII.Invert),
		labelStyle.Render("Export"),
		fmt.Sprintf("%s -> %s", strings.ToUpper(string(m.draft.ExportFormat)), summarizeText(m.draft.ExportOut)),
		"",
		labelStyle.Render("Command Guide"),
		"/image <path>",
		"/text-file <path>",
		"/text \"Happy Birthday\"",
		"/charset \"@#* .\"",
		"/density 96",
		"/threshold 0.42",
		"/export pdf ./exports/out.pdf",
	}

	if len(m.history) > 0 {
		lines = append(lines, "", labelStyle.Render("Recent Commands"))
		for _, command := range tailStrings(m.history, 4) {
			lines = append(lines, truncateForWidth(command, width))
		}
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderCommandPanel() string {
	parts := []string{
		sectionTitleStyle.Render("Command"),
		m.commandInput.View(),
		mutedStyle.Render("Examples: /image ./test.png  ·  /text-file ./message.txt  ·  /export txt ./exports/out.txt"),
	}

	if m.status.Text != "" {
		if m.status.IsError {
			parts = append(parts, errorStyle.Render(m.status.Text))
		} else {
			parts = append(parts, successStyle.Render(m.status.Text))
		}
	}

	return strings.Join(parts, "\n")
}

func importTextFile(path string) (string, error) {
	if err := validateTextFile(path); err != nil {
		return "", err
	}

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("could not read %s: %w", path, err)
	}

	return string(data), nil
}

func validateTextFile(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("usage: /text-file <path>")
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".txt", ".md":
	default:
		return fmt.Errorf("text-file supports only .txt and .md files")
	}

	info, err := os.Stat(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("text file not found: %s", path)
	}
	if info.IsDir() {
		return fmt.Errorf("text-file path must be a file")
	}
	return nil
}

func validateImageFile(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("usage: /image <path>")
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".webp":
	default:
		return fmt.Errorf("image supports .png, .jpg, .jpeg, and .webp")
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

func normalizeGlyphSource(text string) string {
	replacer := strings.NewReplacer("\r", " ", "\n", " ", "\t", " ")
	clean := strings.Join(strings.Fields(replacer.Replace(text)), " ")
	if clean == "" {
		return ""
	}

	runes := []rune(clean)
	if len(runes) > 128 {
		runes = runes[:128]
	}
	if len(runes) == 1 {
		return string(append(runes, ' '))
	}
	return string(runes)
}

func suggestedExportPath(format exportpkg.ASCIIFormat) string {
	return filepath.Join("exports", "ascii-art."+string(format))
}

func summarizeText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Not set"
	}
	if len(value) <= 48 {
		return value
	}
	return value[:45] + "..."
}

func truncateForWidth(text string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= width {
		return text
	}
	if width <= 1 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "…"
}

func tailStrings(values []string, count int) []string {
	if len(values) <= count {
		return values
	}
	return values[len(values)-count:]
}

func formatThreshold(value float64) string {
	if value == 0 {
		return "disabled"
	}
	return fmt.Sprintf("%.2f", value)
}

func (m Model) effectiveDensity() int {
	if m.draft.ASCII.Density > 0 {
		return m.draft.ASCII.Density
	}
	return 80
}

func joinPanels(left, right string, totalWidth int) string {
	if strings.TrimSpace(right) == "" {
		return left
	}
	if totalWidth < 100 {
		return left + "\n\n" + right
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
