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

func TestViewShowsSlashCommandScaffold(t *testing.T) {
	view := NewModel().View()

	for _, fragment := range []string{
		"letterpress",
		"ASCII Preview",
		"Current State",
		"Command",
		"/image <path>",
		"/export pdf",
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

func TestImageCommandUpdatesStateAndPreview(t *testing.T) {
	model := NewModel()
	path := writeImageFixture(t)

	model.commandInput.SetValue("/image " + path)
	model = model.executeCurrentCommand()

	if model.draft.ImagePath != path {
		t.Fatalf("draft.ImagePath = %q, want %q", model.draft.ImagePath, path)
	}
	if model.previewErr != "" {
		t.Fatalf("previewErr = %q", model.previewErr)
	}
	if model.preview.Width == 0 || model.preview.Height == 0 {
		t.Fatalf("expected non-empty preview, got %dx%d", model.preview.Width, model.preview.Height)
	}
}

func TestTextFileCommandLoadsLetteringAndDerivesCharset(t *testing.T) {
	model := NewModel()
	path := writeTextFixture(t, "glyphs.txt", "Happy Birthday")

	model.commandInput.SetValue("/text-file " + path)
	model = model.executeCurrentCommand()

	if model.draft.TextFile != path {
		t.Fatalf("draft.TextFile = %q, want %q", model.draft.TextFile, path)
	}
	if model.draft.Text != "Happy Birthday" {
		t.Fatalf("draft.Text = %q", model.draft.Text)
	}
	if got := model.draft.EffectiveASCIIOptions().Charset; got != "Happy Birthday" {
		t.Fatalf("derived charset = %q, want %q", got, "Happy Birthday")
	}
}

func TestInvalidDensityCommandPreservesExistingState(t *testing.T) {
	model := NewModel()
	model.draft.ASCII.Density = 96

	model.commandInput.SetValue("/density nope")
	model = model.executeCurrentCommand()

	if model.draft.ASCII.Density != 96 {
		t.Fatalf("density changed on invalid command: %d", model.draft.ASCII.Density)
	}
	if !model.status.IsError {
		t.Fatal("expected invalid density to produce an error status")
	}
}

func TestHistoryRecallMovesThroughCommands(t *testing.T) {
	model := NewModel()
	model.pushHistory("/size A5")
	model.pushHistory("/density 64")

	model.historyPrev()
	if got := model.commandInput.Value(); got != "/density 64" {
		t.Fatalf("first historyPrev() = %q, want %q", got, "/density 64")
	}

	model.historyPrev()
	if got := model.commandInput.Value(); got != "/size A5" {
		t.Fatalf("second historyPrev() = %q, want %q", got, "/size A5")
	}

	model.historyNext()
	if got := model.commandInput.Value(); got != "/density 64" {
		t.Fatalf("historyNext() = %q, want %q", got, "/density 64")
	}
}

func TestExportCommandDelegatesToASCIIExporter(t *testing.T) {
	model := NewModel()
	model.draft.ImagePath = "/tmp/test.png"
	model.draft.Text = "HELLO"

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
	if gotASCII.Charset != "HELLO" {
		t.Fatalf("ascii charset = %q, want %q", gotASCII.Charset, "HELLO")
	}
	if gotExport.Format != exportpkg.ASCIIFormatTXT {
		t.Fatalf("export format = %q, want %q", gotExport.Format, exportpkg.ASCIIFormatTXT)
	}
	if gotExport.Out != "./exports/out.txt" {
		t.Fatalf("export out = %q, want %q", gotExport.Out, "./exports/out.txt")
	}
	if gotPage.Size != domain.PageSizeA4 || gotPage.Orientation != domain.OrientationPortrait {
		t.Fatalf("unexpected export page: %+v", gotPage)
	}
	if model.status.IsError {
		t.Fatalf("expected export success, got %q", model.status.Text)
	}
}

func TestRefreshPreviewUsesComposeASCII(t *testing.T) {
	model := NewModel()
	model.draft.ImagePath = "/tmp/image.png"

	origCompose := composeASCII
	defer func() { composeASCII = origCompose }()

	composeASCII = func(page domain.ProjectPage, imagePath string, options domain.ASCIIOptions) (*renderpkg.ASCIIComposition, error) {
		return &renderpkg.ASCIIComposition{
			Page: renderpkg.Page{
				Size:        page.Size,
				Orientation: page.Orientation,
			},
			ImagePath: imagePath,
			Options:   options,
			Art: asciipkg.Art{
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

func writeTextFixture(t *testing.T, name, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
