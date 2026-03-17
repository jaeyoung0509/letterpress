package tui

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	asciipkg "github.com/jaeyoung0509/letterpress/internal/ascii"
	"github.com/jaeyoung0509/letterpress/internal/domain"
	exportpkg "github.com/jaeyoung0509/letterpress/internal/export"
	renderpkg "github.com/jaeyoung0509/letterpress/internal/render"
	"github.com/sahilm/fuzzy"
)

var (
	composeASCII         = renderpkg.ComposeASCII
	composeAndWriteASCII = exportpkg.ComposeAndWriteASCII
)

var commandCatalog = []string{
	"/image ",
	"/text-file ",
	`/text "Happy Birthday"`,
	"/mode ascii",
	"/size A4",
	"/orientation portrait",
	`/charset "@#* ."`,
	"/density 96",
	"/threshold 0.42",
	"/invert on",
	"/export pdf ./exports/out.pdf",
	"/export txt ./exports/out.txt",
	"/help",
}

type KeyMap struct {
	Run          key.Binding
	Autocomplete key.Binding
	HistoryPrev  key.Binding
	HistoryNext  key.Binding
	ScrollUp     key.Binding
	ScrollDown   key.Binding
	PickerPrev   key.Binding
	PickerNext   key.Binding
	Clear        key.Binding
	Quit         key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Run:          key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "run")),
		Autocomplete: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "complete")),
		HistoryPrev:  key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "history prev")),
		HistoryNext:  key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "history next")),
		ScrollUp:     key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "preview up")),
		ScrollDown:   key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdn", "preview down")),
		PickerPrev:   key.NewBinding(key.WithKeys("ctrl+p"), key.WithHelp("ctrl+p", "picker prev")),
		PickerNext:   key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "picker next")),
		Clear:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear")),
		Quit:         key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Run, k.Autocomplete, k.PickerPrev, k.PickerNext, k.ScrollUp, k.ScrollDown, k.Clear, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Run, k.Autocomplete, k.HistoryPrev, k.HistoryNext, k.PickerPrev, k.PickerNext, k.ScrollUp, k.ScrollDown, k.Clear, k.Quit}}
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
	options := d.ASCII
	if strings.TrimSpace(options.Charset) == "" {
		if derived := normalizeGlyphSource(d.Text); derived != "" {
			options.Charset = derived
		}
	}
	return options
}

func (d Draft) GlyphSourceLabel() string {
	if strings.TrimSpace(d.ASCII.Charset) != "" {
		return "explicit charset"
	}
	if strings.TrimSpace(d.Text) != "" {
		return "lettering-derived"
	}
	return "default ascii ramp"
}

type slashCommand struct {
	Name string
	Args []string
}

type suggestion struct {
	Display     string
	Replacement string
}

type pathCandidate struct {
	path string
	dir  bool
}

type Model struct {
	width  int
	height int

	keyMap KeyMap
	help   help.Model

	commandInput textinput.Model
	status       statusMessage

	draft Draft

	workingDir string

	preview    asciipkg.Art
	previewErr string
	viewport   viewport.Model

	suggestions     []suggestion
	suggestionIndex int

	history      []string
	historyIndex int
	historyDraft string
}

func NewModel() Model {
	workingDir, err := os.Getwd()
	if err != nil {
		workingDir = "."
	}

	input := textinput.New()
	input.Prompt = "> "
	input.Placeholder = `/image ./test.png`
	input.Width = 72
	input.Focus()

	model := Model{
		width:        120,
		height:       38,
		keyMap:       DefaultKeyMap(),
		help:         help.New(),
		commandInput: input,
		draft:        newDraft(),
		workingDir:   workingDir,
		historyIndex: -1,
	}
	model.viewport = viewport.New(80, 16)
	model.viewport.MouseWheelEnabled = true
	model.syncLayout()
	model.refreshPreview()
	model.updateSuggestions()
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
		m.syncLayout()
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keyMap.Run):
			m = m.executeCurrentCommand()
			return m, nil
		case key.Matches(msg, m.keyMap.Autocomplete):
			m.applySuggestion()
			return m, nil
		case key.Matches(msg, m.keyMap.HistoryPrev):
			m.historyPrev()
			m.updateSuggestions()
			return m, nil
		case key.Matches(msg, m.keyMap.HistoryNext):
			m.historyNext()
			m.updateSuggestions()
			return m, nil
		case key.Matches(msg, m.keyMap.ScrollUp):
			m.viewport.HalfViewUp()
			return m, nil
		case key.Matches(msg, m.keyMap.ScrollDown):
			m.viewport.HalfViewDown()
			return m, nil
		case key.Matches(msg, m.keyMap.PickerPrev):
			m.moveSuggestion(-1)
			return m, nil
		case key.Matches(msg, m.keyMap.PickerNext):
			m.moveSuggestion(1)
			return m, nil
		case key.Matches(msg, m.keyMap.Clear):
			m.commandInput.SetValue("")
			m.updateSuggestions()
			m.setStatus("Command cleared.", false)
			return m, nil
		}
	}

	var (
		inputCmd    tea.Cmd
		viewportCmd tea.Cmd
	)
	m.commandInput, inputCmd = m.commandInput.Update(msg)
	m.viewport, viewportCmd = m.viewport.Update(msg)
	m.updateSuggestions()
	return m, tea.Batch(inputCmd, viewportCmd)
}

func (m Model) View() string {
	contentWidth := max(56, m.width-appFrameStyle.GetHorizontalFrameSize()-2)
	contentHeight := max(18, m.height-appFrameStyle.GetVerticalFrameSize())

	header := shellCardStyle.Width(contentWidth).Render(m.renderHeaderBody())
	status := m.renderStatusLine(contentWidth)
	prompt := promptPaneStyle.Width(contentWidth).Render(m.renderPromptPane(contentWidth))
	footer := mutedStyle.Width(contentWidth).Render(m.help.View(m.keyMap))

	reservedHeight := lipgloss.Height(header) + lipgloss.Height(status) + lipgloss.Height(prompt) + lipgloss.Height(footer) + 4
	previewBoxHeight := max(6, contentHeight-reservedHeight)
	preview := previewPaneStyle.Width(contentWidth).Height(previewBoxHeight).Render(m.renderPreviewPane(contentWidth, previewBoxHeight))

	body := strings.Join([]string{header, status, preview, prompt, footer}, "\n")
	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, appFrameStyle.Render(body))
}

func (m *Model) syncLayout() {
	availableWidth := max(42, m.width-appFrameStyle.GetHorizontalFrameSize()-6)
	m.commandInput.Width = max(24, availableWidth-4)

	previewHeight := max(8, m.height-18)
	m.viewport.Width = max(30, availableWidth)
	m.viewport.Height = previewHeight
	m.syncPreviewViewport()
}

func (m *Model) syncPreviewViewport() {
	m.viewport.SetContent(m.previewContent(max(20, m.viewport.Width)))
	m.viewport.GotoTop()
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
	m.updateSuggestions()

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
		m.setStatus("Commands: /image /text-file /text /size /orientation /charset /density /threshold /invert /export", false)
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

	target := suggestedExportPath(format)
	if len(args) == 2 {
		target = args[1]
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
		m.previewErr = ""
		m.syncPreviewViewport()
		return
	}

	composition, err := composeASCII(m.draft.Page, m.draft.ImagePath, m.draft.EffectiveASCIIOptions())
	if err != nil {
		m.preview = asciipkg.Art{}
		m.previewErr = err.Error()
		m.syncPreviewViewport()
		return
	}

	m.preview = composition.Art
	m.previewErr = ""
	m.syncPreviewViewport()
}

func (m *Model) updateSuggestions() {
	current := ""
	if len(m.suggestions) > 0 && m.suggestionIndex >= 0 && m.suggestionIndex < len(m.suggestions) {
		current = m.suggestions[m.suggestionIndex].Display
	}
	m.suggestions = suggestForInput(m.commandInput.Value(), m.workingDir)
	if len(m.suggestions) == 0 {
		m.suggestionIndex = 0
		return
	}
	for idx, item := range m.suggestions {
		if item.Display == current {
			m.suggestionIndex = idx
			return
		}
	}
	m.suggestionIndex = 0
}

func (m *Model) applySuggestion() {
	if len(m.suggestions) == 0 {
		return
	}
	index := clamp(m.suggestionIndex, 0, len(m.suggestions)-1)
	m.commandInput.SetValue(m.suggestions[index].Replacement)
	m.commandInput.CursorEnd()
	m.updateSuggestions()
}

func (m *Model) moveSuggestion(delta int) {
	if len(m.suggestions) == 0 {
		return
	}
	m.suggestionIndex += delta
	if m.suggestionIndex < 0 {
		m.suggestionIndex = len(m.suggestions) - 1
	}
	if m.suggestionIndex >= len(m.suggestions) {
		m.suggestionIndex = 0
	}
}

func suggestForInput(input, cwd string) []suggestion {
	trimmed := strings.TrimLeft(input, " ")
	if trimmed == "" || !strings.HasPrefix(trimmed, "/") {
		return nil
	}

	if !strings.Contains(trimmed, " ") {
		return commandSuggestions(trimmed)
	}

	command, args, trailingSpace := splitForSuggestions(trimmed)
	switch command {
	case "/image":
		partial := ""
		if len(args) > 0 {
			partial = args[0]
		} else if !trailingSpace {
			return nil
		}
		return pathSuggestions(cwd, trimmed, partial, []string{".png", ".jpg", ".jpeg", ".webp"})
	case "/text-file":
		partial := ""
		if len(args) > 0 {
			partial = args[0]
		} else if !trailingSpace {
			return nil
		}
		return pathSuggestions(cwd, trimmed, partial, []string{".txt", ".md"})
	case "/export":
		if len(args) == 0 {
			return commandSuggestions(trimmed)
		}
		if len(args) == 1 && !trailingSpace {
			return exportFormatSuggestions(trimmed, args[0])
		}
		format := strings.ToLower(args[0])
		if format != "pdf" && format != "txt" {
			return nil
		}
		partial := ""
		if len(args) > 1 {
			partial = args[1]
		}
		extensions := []string{"." + format}
		return pathSuggestions(cwd, trimmed, partial, extensions)
	default:
		return nil
	}
}

func splitForSuggestions(input string) (string, []string, bool) {
	trailingSpace := strings.HasSuffix(input, " ")
	fields := strings.Fields(input)
	if len(fields) == 0 {
		return "", nil, trailingSpace
	}
	if trailingSpace {
		return fields[0], fields[1:], true
	}
	return fields[0], fields[1:], false
}

func commandSuggestions(partial string) []suggestion {
	out := make([]suggestion, 0)
	for _, candidate := range commandCatalog {
		if strings.HasPrefix(candidate, partial) {
			out = append(out, suggestion{
				Display:     candidate,
				Replacement: candidate,
			})
		}
	}
	return out
}

func exportFormatSuggestions(input, partial string) []suggestion {
	options := []string{"pdf", "txt"}
	out := make([]suggestion, 0, len(options))
	for _, candidate := range options {
		if strings.HasPrefix(candidate, strings.ToLower(partial)) {
			out = append(out, suggestion{
				Display:     candidate,
				Replacement: "/export " + candidate + " ",
			})
		}
	}
	return out
}

func pathSuggestions(cwd, input, partial string, allowedExts []string) []suggestion {
	candidates := suggestPathCandidates(cwd, partial, allowedExts)
	out := make([]suggestion, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, suggestion{
			Display:     candidate,
			Replacement: replaceLastToken(input, partial, candidate),
		})
	}
	return out
}

func suggestPathCandidates(cwd, partial string, allowedExts []string) []string {
	partial = strings.Trim(partial, `"`)

	typedDir := "."
	prefix := partial
	if partial != "" {
		typedDir = filepath.Dir(partial)
		prefix = filepath.Base(partial)
	}
	if partial == "" || strings.HasSuffix(partial, string(os.PathSeparator)) {
		typedDir = partial
		prefix = ""
	}
	if typedDir == "" {
		typedDir = "."
	}

	searchDir := typedDir
	if !filepath.IsAbs(searchDir) {
		searchDir = filepath.Join(cwd, searchDir)
	}
	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return nil
	}

	allowed := map[string]struct{}{}
	for _, ext := range allowedExts {
		allowed[strings.ToLower(ext)] = struct{}{}
	}

	candidates := make([]pathCandidate, 0)
	seen := map[string]struct{}{}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
			continue
		}

		candidatePath := displaySuggestionPath(typedDir, name)
		if entry.IsDir() {
			candidatePath += "/"
		}

		if _, ok := seen[candidatePath]; ok {
			continue
		}
		seen[candidatePath] = struct{}{}

		if entry.IsDir() {
			candidates = append(candidates, pathCandidate{path: candidatePath, dir: true})
			continue
		}

		if len(allowed) > 0 {
			if _, ok := allowed[strings.ToLower(filepath.Ext(name))]; !ok {
				continue
			}
		}
		candidates = append(candidates, pathCandidate{path: candidatePath})
	}

	if partial != "" && !filepath.IsAbs(partial) {
		for _, extra := range fuzzyWorkspaceCandidates(cwd, partial, allowed) {
			if _, ok := seen[extra.path]; ok {
				continue
			}
			seen[extra.path] = struct{}{}
			candidates = append(candidates, extra)
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].dir == candidates[j].dir {
			return candidates[i].path < candidates[j].path
		}
		return candidates[i].dir
	})

	out := make([]string, len(candidates))
	for i, candidate := range candidates {
		out[i] = candidate.path
	}
	return out
}

func displaySuggestionPath(baseDir, name string) string {
	if filepath.IsAbs(name) {
		return filepath.ToSlash(name)
	}

	base := strings.TrimSpace(baseDir)
	switch base {
	case "", ".":
		return "./" + filepath.ToSlash(name)
	default:
		clean := filepath.ToSlash(filepath.Join(base, name))
		if strings.HasPrefix(clean, "./") || strings.HasPrefix(clean, "../") {
			return clean
		}
		return "./" + clean
	}
}

func fuzzyWorkspaceCandidates(cwd, partial string, allowed map[string]struct{}) []pathCandidate {
	query := strings.ToLower(strings.TrimSpace(filepath.Base(partial)))
	if query == "" {
		return nil
	}

	all := make([]pathCandidate, 0, 32)
	_ = filepath.WalkDir(cwd, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if path == cwd {
			return nil
		}

		rel, err := filepath.Rel(cwd, path)
		if err != nil {
			return nil
		}

		depth := strings.Count(rel, string(os.PathSeparator))
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".github", "vendor", "node_modules":
				return filepath.SkipDir
			}
			if depth > 3 {
				return filepath.SkipDir
			}
		} else if depth > 4 {
			return nil
		}

		display := displaySuggestionPath(filepath.Dir(rel), entry.Name())
		name := strings.ToLower(entry.Name())
		full := strings.ToLower(filepath.ToSlash(rel))
		if !strings.Contains(name, query) && !strings.Contains(full, query) {
			if len(query) < 2 {
				return nil
			}
		}

		if entry.IsDir() {
			all = append(all, pathCandidate{path: display + "/", dir: true})
			return nil
		}

		if len(allowed) > 0 {
			if _, ok := allowed[strings.ToLower(filepath.Ext(entry.Name()))]; !ok {
				return nil
			}
		}
		all = append(all, pathCandidate{path: display})
		return nil
	})

	if len(all) == 0 {
		return nil
	}

	index := make([]string, len(all))
	for i, item := range all {
		index[i] = strings.ToLower(strings.TrimPrefix(item.path, "./"))
	}

	matches := fuzzy.Find(query, index)
	if len(matches) == 0 {
		sort.Slice(all, func(i, j int) bool {
			if all[i].dir == all[j].dir {
				return all[i].path < all[j].path
			}
			return all[i].dir
		})
		if len(all) > 12 {
			all = all[:12]
		}
		return all
	}

	out := make([]pathCandidate, 0, min(12, len(matches)))
	for _, match := range matches[:min(12, len(matches))] {
		out = append(out, all[match.Index])
	}
	return out
}

func replaceLastToken(input, partial, replacement string) string {
	if partial == "" {
		if strings.HasSuffix(input, " ") {
			return input + replacement
		}
		return input + " " + replacement
	}
	index := strings.LastIndex(input, partial)
	if index == -1 {
		return input
	}
	return input[:index] + replacement
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

func (m Model) renderHeaderBody() string {
	sources := []string{
		fmt.Sprintf("image %s", summarizeText(m.draft.ImagePath)),
		fmt.Sprintf("text %s", summarizeText(m.draft.TextFile)),
		fmt.Sprintf("glyphs %s", m.draft.GlyphSourceLabel()),
	}
	if strings.TrimSpace(m.draft.TextFile) == "" && strings.TrimSpace(m.draft.Text) != "" {
		sources[1] = fmt.Sprintf("text %s", summarizeText(m.draft.Text))
	}

	lines := []string{
		headerStyle.Render(">_ letterpress ascii"),
		mutedStyle.Render(fmt.Sprintf("%s %s  ·  %s  ·  density %d  ·  threshold %s  ·  invert %t",
			m.draft.Page.Size,
			m.draft.Page.Orientation,
			strings.ToUpper(string(m.draft.ExportFormat)),
			effectiveDensity(m.draft.ASCII.Density),
			formatThreshold(m.draft.ASCII.Threshold),
			m.draft.ASCII.Invert,
		)),
		mutedStyle.Render(strings.Join(sources, "  ·  ")),
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderStatusLine(width int) string {
	style := mutedStyle.Width(width)
	if m.status.Text == "" {
		return style.Render("Tip: Tab completes commands and paths. Use PgUp/PgDn to scroll the preview while the header and prompt stay fixed.")
	}
	if m.status.IsError {
		return errorStyle.Width(width).Render(m.status.Text)
	}
	return successStyle.Width(width).Render(m.status.Text)
}

func (m Model) renderPreviewPane(width, boxHeight int) string {
	viewportWidth := max(24, width-previewPaneStyle.GetHorizontalFrameSize())
	viewportHeight := max(4, boxHeight-previewPaneStyle.GetVerticalFrameSize()-1)

	localViewport := m.viewport
	localViewport.Width = viewportWidth
	localViewport.Height = viewportHeight
	localViewport.SetContent(m.previewContent(viewportWidth))

	meta := "  setup"
	if len(m.preview.Lines) > 0 {
		meta = fmt.Sprintf("  %d lines · scroll %d%%", len(m.preview.Lines), int(localViewport.ScrollPercent()*100))
	}
	title := sectionTitleStyle.Render("Preview") + mutedStyle.Render(meta)
	if m.previewErr != "" {
		return strings.Join([]string{
			title,
			errorStyle.Render(m.previewErr),
		}, "\n")
	}
	return strings.Join([]string{
		title,
		localViewport.View(),
	}, "\n")
}

func (m Model) previewContent(width int) string {
	if strings.TrimSpace(m.draft.ImagePath) == "" {
		lines := []string{
			"No source image configured.",
			"",
			"1. /image ./test.png",
			"2. /text-file ./test.txt",
			"3. /export txt ./exports/out.txt",
			"",
			"Tab completes matching commands and file paths.",
		}
		return strings.Join(lines, "\n")
	}
	if len(m.preview.Lines) == 0 {
		return "(preview pending)"
	}

	lines := make([]string, 0, len(m.preview.Lines))
	for _, line := range m.preview.Lines {
		lines = append(lines, truncateRunes(line, width))
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderPromptPane(width int) string {
	lines := []string{m.commandInput.View()}

	if len(m.suggestions) > 0 {
		lines = append(lines, mutedStyle.Render("picker"))
		for idx, item := range m.visibleSuggestions(6) {
			if idx == m.suggestionIndex-m.suggestionWindowStart(6) {
				lines = append(lines, pickerActiveStyle.Render("› "+item.Display))
				continue
			}
			lines = append(lines, pickerItemStyle.Render("  "+item.Display))
		}
	} else {
		lines = append(lines, mutedStyle.Render("picker"), mutedStyle.Render("  no suggestions"))
	}

	lines = append(lines, mutedStyle.Width(width).Render(`examples: /image test  ·  /text-file test  ·  /export pdf ./exports/out.pdf`))
	return strings.Join(lines, "\n")
}

func (m Model) suggestionWindowStart(limit int) int {
	if len(m.suggestions) <= limit {
		return 0
	}
	index := clamp(m.suggestionIndex, 0, len(m.suggestions)-1)
	start := index - limit/2
	if start < 0 {
		return 0
	}
	maxStart := len(m.suggestions) - limit
	if start > maxStart {
		return maxStart
	}
	return start
}

func (m Model) visibleSuggestions(limit int) []suggestion {
	if len(m.suggestions) == 0 {
		return nil
	}
	start := m.suggestionWindowStart(limit)
	end := min(len(m.suggestions), start+limit)
	return m.suggestions[start:end]
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
		return "not set"
	}
	if len([]rune(value)) <= 54 {
		return value
	}
	return truncateRunes(value, 54)
}

func centerText(text string, width int) string {
	runes := []rune(text)
	if len(runes) >= width {
		return truncateRunes(text, width)
	}
	padding := width - len(runes)
	left := padding / 2
	right := padding - left
	return strings.Repeat(" ", left) + text + strings.Repeat(" ", right)
}

func truncateRunes(text string, width int) string {
	runes := []rune(text)
	if len(runes) <= width {
		return text
	}
	if width <= 1 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "…"
}

func formatThreshold(value float64) string {
	if value == 0 {
		return "disabled"
	}
	return fmt.Sprintf("%.2f", value)
}

func effectiveDensity(value int) int {
	if value > 0 {
		return value
	}
	return 80
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

func clamp(value, lower, upper int) int {
	if value < lower {
		return lower
	}
	if value > upper {
		return upper
	}
	return value
}
