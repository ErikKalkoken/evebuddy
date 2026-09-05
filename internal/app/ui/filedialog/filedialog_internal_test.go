package filedialog

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// UIServiceFake is a minimal baseUI implementation for unit tests.
type UIServiceFake struct {
	app        fyne.App
	mainWindow fyne.Window

	isMobile bool

	getOrCreateWindowCalls int
	lastWindowID           string
	lastTitle              string
	lastCreatedWindow       fyne.Window
}

func newUIServiceFake(a fyne.App) *UIServiceFake {
	return &UIServiceFake{app: a, mainWindow: a.NewWindow("Main")}
}

func (u *UIServiceFake) ErrorDisplay(err error) string { return err.Error() }

func (u *UIServiceFake) GetOrCreateWindow(id string, titles ...string) (fyne.Window, bool) {
	u.getOrCreateWindowCalls++
	u.lastWindowID = id
	u.lastTitle = strings.Join(titles, " - ")
	u.lastCreatedWindow = u.app.NewWindow(u.lastTitle)
	return u.lastCreatedWindow, true
}

func (u *UIServiceFake) IsDeveloperMode() bool   { return false }
func (u *UIServiceFake) IsMobile() bool          { return u.isMobile }
func (u *UIServiceFake) MainWindow() fyne.Window { return u.mainWindow }

var _ baseUI = (*UIServiceFake)(nil)

func TestHandleSaveResult(t *testing.T) {
	t.Run("shows error and skips write func when the dialog reports an error", func(t *testing.T) {
		u := newUIServiceFake(test.NewTempApp(t))
		msgs := make(chan string, 1)
		called := false
		arg := ShowFileSaveWindowParams{
			ShowSnackbar: func(s string) { msgs <- s },
			WriteFunc: func(context.Context, io.Writer) error {
				called = true
				return nil
			},
		}

		handleSaveResult(u, arg, nil, errors.New("boom"))

		assert.False(t, called)
		select {
		case msg := <-msgs:
			assert.Equal(t, "Error: boom", msg)
		default:
			t.Fatal("expected a snackbar message")
		}
	})

	t.Run("silently ignores a nil writer with no error", func(t *testing.T) {
		u := newUIServiceFake(test.NewTempApp(t))
		called := false
		arg := ShowFileSaveWindowParams{
			ShowSnackbar: func(s string) { t.Fatalf("unexpected snackbar: %s", s) },
			WriteFunc: func(context.Context, io.Writer) error {
				called = true
				return nil
			},
		}

		handleSaveResult(u, arg, nil, nil)

		assert.False(t, called)
	})

	t.Run("writes the file and shows completion text on success", func(t *testing.T) {
		u := newUIServiceFake(test.NewTempApp(t))
		path := filepath.Join(t.TempDir(), "out.txt")
		writer, err := storage.SaveFileToURI(storage.NewFileURI(path))
		require.NoError(t, err)
		msgs := make(chan string, 1)
		arg := ShowFileSaveWindowParams{
			CompletionText: "done",
			ShowSnackbar:   func(s string) { msgs <- s },
			WriteFunc: func(_ context.Context, w io.Writer) error {
				_, err := w.Write([]byte("hello"))
				return err
			},
		}

		handleSaveResult(u, arg, writer, nil)

		select {
		case msg := <-msgs:
			assert.Equal(t, "done", msg)
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for snackbar")
		}
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "hello", string(data))
	})

	t.Run("shows error text when write func fails", func(t *testing.T) {
		u := newUIServiceFake(test.NewTempApp(t))
		path := filepath.Join(t.TempDir(), "out.txt")
		writer, err := storage.SaveFileToURI(storage.NewFileURI(path))
		require.NoError(t, err)
		msgs := make(chan string, 1)
		arg := ShowFileSaveWindowParams{
			ShowSnackbar: func(s string) { msgs <- s },
			WriteFunc: func(context.Context, io.Writer) error {
				return errors.New("disk full")
			},
		}

		handleSaveResult(u, arg, writer, nil)

		select {
		case msg := <-msgs:
			assert.Equal(t, "Error: disk full", msg)
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for snackbar")
		}
	})
}

func TestHandleOpenResult(t *testing.T) {
	t.Run("shows error and skips read func when the dialog reports an error", func(t *testing.T) {
		u := newUIServiceFake(test.NewTempApp(t))
		msgs := make(chan string, 1)
		called := false
		arg := ShowFileOpenWindowParams{
			ShowSnackbar: func(s string) { msgs <- s },
			ReadFunc: func(context.Context, io.Reader) error {
				called = true
				return nil
			},
		}

		handleOpenResult(u, arg, nil, errors.New("boom"))

		assert.False(t, called)
		select {
		case msg := <-msgs:
			assert.Equal(t, "Error: boom", msg)
		default:
			t.Fatal("expected a snackbar message")
		}
	})

	t.Run("silently ignores a nil reader with no error", func(t *testing.T) {
		u := newUIServiceFake(test.NewTempApp(t))
		called := false
		arg := ShowFileOpenWindowParams{
			ShowSnackbar: func(s string) { t.Fatalf("unexpected snackbar: %s", s) },
			ReadFunc: func(context.Context, io.Reader) error {
				called = true
				return nil
			},
		}

		handleOpenResult(u, arg, nil, nil)

		assert.False(t, called)
	})

	t.Run("reads the file and shows completion text on success", func(t *testing.T) {
		u := newUIServiceFake(test.NewTempApp(t))
		path := filepath.Join(t.TempDir(), "in.txt")
		require.NoError(t, os.WriteFile(path, []byte("hello"), 0o600))
		reader, err := storage.OpenFileFromURI(storage.NewFileURI(path))
		require.NoError(t, err)
		msgs := make(chan string, 1)
		var got []byte
		arg := ShowFileOpenWindowParams{
			CompletionText: "done",
			ShowSnackbar:   func(s string) { msgs <- s },
			ReadFunc: func(_ context.Context, r io.Reader) error {
				var err error
				got, err = io.ReadAll(r)
				return err
			},
		}

		handleOpenResult(u, arg, reader, nil)

		select {
		case msg := <-msgs:
			assert.Equal(t, "done", msg)
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for snackbar")
		}
		assert.Equal(t, "hello", string(got))
	})

	t.Run("shows error text when read func fails", func(t *testing.T) {
		u := newUIServiceFake(test.NewTempApp(t))
		path := filepath.Join(t.TempDir(), "in.txt")
		require.NoError(t, os.WriteFile(path, []byte("hello"), 0o600))
		reader, err := storage.OpenFileFromURI(storage.NewFileURI(path))
		require.NoError(t, err)
		msgs := make(chan string, 1)
		arg := ShowFileOpenWindowParams{
			ShowSnackbar: func(s string) { msgs <- s },
			ReadFunc: func(context.Context, io.Reader) error {
				return errors.New("corrupt")
			},
		}

		handleOpenResult(u, arg, reader, nil)

		select {
		case msg := <-msgs:
			assert.Equal(t, "Error: corrupt", msg)
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for snackbar")
		}
	})
}

func TestShowSave_Mobile_SkipsWindowCreation(t *testing.T) {
	a := test.NewTempApp(t)
	u := newUIServiceFake(a)
	u.isMobile = true
	arg := ShowFileSaveWindowParams{
		ShowSnackbar: func(string) {},
		WriteFunc:    func(context.Context, io.Writer) error { return nil },
		Window:       a.NewWindow("parent"),
	}

	assert.NotPanics(t, func() { ShowSave(u, arg) })
	assert.Equal(t, 0, u.getOrCreateWindowCalls)
}

func TestShowOpen_Mobile_SkipsWindowCreation(t *testing.T) {
	a := test.NewTempApp(t)
	u := newUIServiceFake(a)
	u.isMobile = true
	arg := ShowFileOpenWindowParams{
		ShowSnackbar: func(string) {},
		ReadFunc:     func(context.Context, io.Reader) error { return nil },
		Window:       a.NewWindow("parent"),
	}

	assert.NotPanics(t, func() { ShowOpen(u, arg) })
	assert.Equal(t, 0, u.getOrCreateWindowCalls)
}

func TestShowSave_Desktop_UsesWindowIDAndTitle(t *testing.T) {
	a := test.NewTempApp(t)
	u := newUIServiceFake(a)
	arg := ShowFileSaveWindowParams{
		WindowID:     "export",
		Title:        "Export data",
		ShowSnackbar: func(string) {},
		WriteFunc:    func(context.Context, io.Writer) error { return nil },
	}

	ShowSave(u, arg)

	require.Equal(t, 1, u.getOrCreateWindowCalls)
	assert.Equal(t, "export", u.lastWindowID)
	assert.Equal(t, "Export data", u.lastTitle)
}

func TestShowDialogInWindow_ResizesToPercentOfMainWindowAndFixesSize(t *testing.T) {
	a := test.NewTempApp(t)
	u := newUIServiceFake(a)
	u.mainWindow.Resize(fyne.NewSize(1000, 800))
	w := a.NewWindow("dialog host")
	d := dialog.NewFileSave(func(fyne.URIWriteCloser, error) {}, w)

	showDialogInWindow(u, w, d)

	assert.Equal(t, fyne.NewSize(800, 640), w.Canvas().Size())
	fs, ok := w.(interface{ FixedSize() bool })
	require.True(t, ok)
	assert.True(t, fs.FixedSize())
}

func TestShowDialogInWindow_ClosingDialogClosesWindow(t *testing.T) {
	a := test.NewTempApp(t)
	u := newUIServiceFake(a)
	w := a.NewWindow("dialog host")
	d := dialog.NewFileSave(func(fyne.URIWriteCloser, error) {}, w)

	showDialogInWindow(u, w, d)
	before := len(a.Driver().AllWindows())
	d.Hide()
	after := len(a.Driver().AllWindows())

	assert.Less(t, after, before)
}

func TestCreateFileSaveDialog_AppliesFilenameAndExtensions(t *testing.T) {
	a := test.NewTempApp(t)
	u := newUIServiceFake(a)
	w := a.NewWindow("host")
	arg := ShowFileSaveWindowParams{
		Filename:     "export.json",
		Extensions:   []string{".json"},
		ShowSnackbar: func(string) {},
		WriteFunc:    func(context.Context, io.Writer) error { return nil },
	}

	var d *dialog.FileDialog
	assert.NotPanics(t, func() { d = createFileSaveDialog(u, arg, w) })
	assert.NotNil(t, d)
}

func TestCreateFileOpenDialog_AppliesExtensions(t *testing.T) {
	a := test.NewTempApp(t)
	u := newUIServiceFake(a)
	w := a.NewWindow("host")
	arg := ShowFileOpenWindowParams{
		Extensions:   []string{".json"},
		ShowSnackbar: func(string) {},
		ReadFunc:     func(context.Context, io.Reader) error { return nil },
	}

	var d *dialog.FileDialog
	assert.NotPanics(t, func() { d = createFileOpenDialog(u, arg, w) })
	assert.NotNil(t, d)
}
