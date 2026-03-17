package tui

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	asciipkg "github.com/jaeyoung0509/letterpress/internal/ascii"
	"github.com/jaeyoung0509/letterpress/internal/domain"
	exportpkg "github.com/jaeyoung0509/letterpress/internal/export"
	renderpkg "github.com/jaeyoung0509/letterpress/internal/render"
)

func TestViewShowsShellScaffold(t *testing.T) {
	view := NewModel().View()

	for _, fragment := range []string{
		">_ letterpress ascii",
		"Preview",
		"picker",
		"Tab completes commands and paths",
	} {
		if !strings.Contains(view, fragment) {
			t.Fatalf("expected view to contain %q, got %q", fragment, view)
		}
	}
}

func TestParseSlashCommandSupportsQuotedArguments(t *testing.T) {
	command, err := parseSlashCommand(`/charset "@#S%?*+;:,. "`)
	if err != nil {
		t.Fatalf("parseSlashCommand() error = %v", err)
	}

	if command.Name != "charset" {
		t.Fatalf("command.Name = %q, want %q", command.Name, "charset")
	}
	if len(command.Args) != 1 || command.Args[0] != "@#S%?*+;:,. " {
		t.Fatalf("command.Args = %#v", command.Args)
	}
}

func TestSuggestPathCandidatesFiltersImageExtensions(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.png"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "assets"), 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	got := suggestPathCandidates(root, "", []string{".png", ".jpg"})
	want := []string{"./assets/", "./a.png"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("suggestPathCandidates() = %v, want %v", got, want)
	}
}

func TestSuggestPathCandidatesSearchesWorkspaceRecursively(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "nested", "images"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	target := filepath.Join(root, "nested", "images", "birthday-card.png")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got := suggestPathCandidates(root, "birth", []string{".png"})
	if len(got) == 0 {
		t.Fatal("expected recursive suggestions")
	}
	if got[0] != "./nested/images/birthday-card.png" {
		t.Fatalf("suggestPathCandidates()[0] = %q", got[0])
	}
}

func TestSuggestPathCandidatesUsesFuzzyMatching(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "nested", "images"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	target := filepath.Join(root, "nested", "images", "birthday-card.png")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got := suggestPathCandidates(root, "btdcrd", []string{".png"})
	if len(got) == 0 {
		t.Fatal("expected fuzzy suggestions")
	}
	if got[0] != "./nested/images/birthday-card.png" {
		t.Fatalf("suggestPathCandidates()[0] = %q", got[0])
	}
}

func TestAutocompleteAppliesTopSuggestion(t *testing.T) {
	root := t.TempDir()
	imagePath := filepath.Join(root, "test-image.png")
	if err := os.WriteFile(imagePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	model := NewModel()
	model.commandInput.SetValue("/image " + filepath.ToSlash(filepath.Join(root, "test")))
	model.updateSuggestions()
	if len(model.suggestions) == 0 {
		t.Fatal("expected suggestions")
	}

	model.applySuggestion()

	if got := model.commandInput.Value(); !strings.Contains(got, "test-image.png") {
		t.Fatalf("command input = %q", got)
	}
}

func TestPickerSelectionMovesAndApplies(t *testing.T) {
	model := NewModel()
	model.suggestions = []suggestion{
		{Display: "./one.png", Replacement: "/image ./one.png"},
		{Display: "./two.png", Replacement: "/image ./two.png"},
	}

	model.moveSuggestion(1)
	model.applySuggestion()

	if got := model.commandInput.Value(); got != "/image ./two.png" {
		t.Fatalf("command input = %q", got)
	}
}

func TestImageCommandUpdatesPreview(t *testing.T) {
	model := NewModel()
	path := writeImageFixture(t)

	model.commandInput.SetValue("/image " + path)
	model = model.executeCurrentCommand()

	if model.draft.ImagePath != path {
		t.Fatalf("draft.ImagePath = %q, want %q", model.draft.ImagePath, path)
	}
	if model.preview.Width == 0 || model.preview.Height == 0 {
		t.Fatalf("expected preview to be populated, got %dx%d", model.preview.Width, model.preview.Height)
	}
}

func TestFillFileCommandLoadsFillText(t *testing.T) {
	model := NewModel()
	path := filepath.Join(t.TempDir(), "message.txt")
	if err := os.WriteFile(path, []byte("Happy Birthday\nFrom Letterpress"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	model.commandInput.SetValue("/fill-file " + path)
	model = model.executeCurrentCommand()

	if model.draft.TextFile != path {
		t.Fatalf("draft.TextFile = %q, want %q", model.draft.TextFile, path)
	}
	if model.draft.Text != "Happy Birthday From Letterpress" {
		t.Fatalf("draft.Text = %q", model.draft.Text)
	}
}

func TestModeCommandAcceptsVectorMode(t *testing.T) {
	model := NewModel()
	model.commandInput.SetValue("/mode vector")
	model = model.executeCurrentCommand()

	if model.draft.ASCII.Mode != domain.ASCIIModeVector {
		t.Fatalf("mode = %q", model.draft.ASCII.Mode)
	}
}

func TestFillFontCommandUpdatesASCIIStyle(t *testing.T) {
	model := NewModel()
	model.commandInput.SetValue("/fill-font block")
	model = model.executeCurrentCommand()

	if model.draft.ASCII.FillFont != domain.FillFontBlock {
		t.Fatalf("fill font = %q", model.draft.ASCII.FillFont)
	}
}

func TestHeaderAndPromptRemainVisibleWithPreviewContent(t *testing.T) {
	model := NewModel()
	model.width = 100
	model.height = 24
	model.preview = asciipkg.Art{
		Width:  80,
		Height: 60,
		Lines:  make([]string, 60),
	}
	for i := range model.preview.Lines {
		model.preview.Lines[i] = strings.Repeat("@", 80)
	}
	model.syncLayout()
	view := model.View()

	if !strings.Contains(view, ">_ letterpress ascii") {
		t.Fatalf("expected fixed header in view, got %q", view)
	}
	if !strings.Contains(view, "> ") {
		t.Fatalf("expected prompt in view, got %q", view)
	}
}

func TestExportCommandDelegatesToASCIIExporter(t *testing.T) {
	model := NewModel()
	model.draft.ImagePath = "/tmp/test.png"
	model.draft.Text = "HELLO"
	model.draft.ASCII.Mode = domain.ASCIIModeHybrid

	origExporter := composeAndWriteASCII
	defer func() { composeAndWriteASCII = origExporter }()

	var (
		gotPage   domain.ProjectPage
		gotImage  string
		gotASCII  domain.ASCIIOptions
		gotExport exportpkg.ASCIIExportOptions
	)
	composeAndWriteASCII = func(page domain.ProjectPage, imagePath string, asciiOptions domain.ASCIIOptions, exportOptions exportpkg.ASCIIExportOptions) (string, error) {
		gotPage = page
		gotImage = imagePath
		gotASCII = asciiOptions
		gotExport = exportOptions
		return exportOptions.Out, nil
	}

	model.commandInput.SetValue("/export txt ./exports/out.txt")
	model = model.executeCurrentCommand()

	if gotImage != "/tmp/test.png" {
		t.Fatalf("image path = %q, want %q", gotImage, "/tmp/test.png")
	}
	if gotASCII.FillText != "HELLO" {
		t.Fatalf("fill text = %q, want %q", gotASCII.FillText, "HELLO")
	}
	if gotASCII.FillFont != domain.FillFontPlain {
		t.Fatalf("fill font = %q, want %q", gotASCII.FillFont, domain.FillFontPlain)
	}
	if gotASCII.EffectiveToneCharset() == "HELLO" {
		t.Fatalf("tone charset should not be derived from fill text")
	}
	if gotExport.Format != exportpkg.ASCIIFormatTXT {
		t.Fatalf("export format = %q, want %q", gotExport.Format, exportpkg.ASCIIFormatTXT)
	}
	if gotPage.Size != domain.PageSizeA4 || gotPage.Orientation != domain.OrientationPortrait {
		t.Fatalf("unexpected page: %+v", gotPage)
	}
}

func TestRefreshPreviewUsesComposeASCII(t *testing.T) {
	model := NewModel()
	model.draft.ImagePath = "/tmp/image.png"

	origCompose := composeASCII
	defer func() { composeASCII = origCompose }()

	composeASCII = func(page domain.ProjectPage, imagePath string, options domain.ASCIIOptions) (*renderpkg.ASCIIComposition, error) {
		return &renderpkg.ASCIIComposition{
			Page: renderpkg.Page{Size: page.Size, Orientation: page.Orientation},
			Art: asciipkg.Art{
				Mode:   domain.ASCIIModeTone,
				Width:  3,
				Height: 1,
				Lines:  []string{"ABC"},
			},
		}, nil
	}

	model.refreshPreview()
	if model.preview.String() != "ABC" {
		t.Fatalf("preview = %q, want %q", model.preview.String(), "ABC")
	}
}

func writeImageFixture(t *testing.T) string {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if x < 2 {
				img.Set(x, y, color.NRGBA{A: 255})
				continue
			}
			img.Set(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}

	path := filepath.Join(t.TempDir(), "fixture.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}
	return path
}
