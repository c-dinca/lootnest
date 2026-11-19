//go:build linux

// Package desk is what Lootnest asks of a Linux desktop besides its window
// (internal/webkit): dialogs, the file picker, notifications, the clipboard,
// keys kept in the Secret Service, and the tray icon. internal/plat's Linux
// files call it.
//
// GTK may only be used on the main thread. OnMain runs a function there and
// waits for it, from any goroutine: directly when the caller is the main
// goroutine (locked to the main thread from the start), else through the
// GTK loop.
package desk

/*
#cgo pkg-config: gtk+-3.0 libsecret-1
#cgo LDFLAGS: -ldl
#include <stdlib.h>
#include <stdint.h>
#include <dlfcn.h>
#include <gtk/gtk.h>
#include <libsecret/secret.h>

extern gboolean deskIdle(uintptr_t fn);
extern void deskTrayPick(uintptr_t tray, int id);

static gboolean desk_idle(gpointer fn) { return deskIdle((uintptr_t)fn); }
static int desk_init(void) { return gtk_init_check(NULL, NULL); }
static void desk_main(void) { gtk_main(); }
static gboolean desk_quit_cb(gpointer d) { gtk_main_quit(); return FALSE; }
static void desk_quit(void) { g_idle_add(desk_quit_cb, NULL); }
static char *desk_clip_get(void) { return gtk_clipboard_wait_for_text(gtk_clipboard_get(GDK_SELECTION_CLIPBOARD)); }
static void desk_dispatch(uintptr_t fn) { g_idle_add(desk_idle, (gpointer)fn); }

static GtkWindow *desk_parent = NULL;
static void desk_set_parent(void *w) { desk_parent = (GtkWindow *)w; }

// ---------------------------------------------------------------- dialogs

static int desk_message(const char *title, const char *text, int type, int buttons, int default_no) {
	GtkWidget *d = gtk_message_dialog_new(desk_parent, GTK_DIALOG_MODAL | GTK_DIALOG_DESTROY_WITH_PARENT,
		(GtkMessageType)type, GTK_BUTTONS_NONE, "%s", text);
	gtk_window_set_title(GTK_WINDOW(d), title);
	// 0 OK, 1 OK and Cancel, 2 Yes and No, 3 Yes, No and Cancel.
	switch (buttons) {
	case 1:
		gtk_dialog_add_buttons(GTK_DIALOG(d), "Cancel", GTK_RESPONSE_CANCEL, "OK", GTK_RESPONSE_OK, NULL);
		break;
	case 2:
		gtk_dialog_add_buttons(GTK_DIALOG(d), "No", GTK_RESPONSE_NO, "Yes", GTK_RESPONSE_YES, NULL);
		break;
	case 3:
		gtk_dialog_add_buttons(GTK_DIALOG(d), "Cancel", GTK_RESPONSE_CANCEL, "No", GTK_RESPONSE_NO, "Yes", GTK_RESPONSE_YES, NULL);
		break;
	default:
		gtk_dialog_add_buttons(GTK_DIALOG(d), "OK", GTK_RESPONSE_OK, NULL);
	}
	if (buttons == 2 || buttons == 3) {
		gtk_dialog_set_default_response(GTK_DIALOG(d), default_no ? GTK_RESPONSE_NO : GTK_RESPONSE_YES);
	}
	int r = gtk_dialog_run(GTK_DIALOG(d));
	gtk_widget_destroy(d);
	return r;
}

// The desktop's own picker (through the portal where there is one).
// Patterns are ';'-separated, as Windows' filters are.
static char *desk_browse(const char *title, int folder, const char *start, const char **names, const char **specs, int n) {
	GtkFileChooserNative *c = gtk_file_chooser_native_new(title, desk_parent,
		folder ? GTK_FILE_CHOOSER_ACTION_SELECT_FOLDER : GTK_FILE_CHOOSER_ACTION_OPEN, "Choose", "Cancel");
	if (start && *start) gtk_file_chooser_set_current_folder(GTK_FILE_CHOOSER(c), start);
	for (int i = 0; i < n; i++) {
		GtkFileFilter *f = gtk_file_filter_new();
		gtk_file_filter_set_name(f, names[i]);
		gchar **pats = g_strsplit(specs[i], ";", -1);
		for (gchar **p = pats; *p; p++) {
			gchar *t = g_strstrip(*p);
			if (*t) gtk_file_filter_add_pattern(f, t);
		}
		g_strfreev(pats);
		gtk_file_chooser_add_filter(GTK_FILE_CHOOSER(c), f);
	}
	char *out = NULL;
	if (gtk_native_dialog_run(GTK_NATIVE_DIALOG(c)) == GTK_RESPONSE_ACCEPT) {
		out = gtk_file_chooser_get_filename(GTK_FILE_CHOOSER(c));
	}
	g_object_unref(c);
	return out;
}

// ---------------------------------------------------------- notifications

// org.freedesktop.Notifications, asynchronously; nothing waits for it.
static void desk_notify(const char *title, const char *body, const char *icon) {
	GDBusConnection *bus = g_bus_get_sync(G_BUS_TYPE_SESSION, NULL, NULL);
	if (!bus) return;
	GVariantBuilder actions, hints;
	g_variant_builder_init(&actions, G_VARIANT_TYPE("as"));
	g_variant_builder_init(&hints, G_VARIANT_TYPE("a{sv}"));
	g_variant_builder_add(&hints, "{sv}", "desktop-entry", g_variant_new_string("lootnest"));
	g_dbus_connection_call(bus, "org.freedesktop.Notifications", "/org/freedesktop/Notifications",
		"org.freedesktop.Notifications", "Notify",
		g_variant_new("(susssasa{sv}i)", "Lootnest", 0, icon, title, body, &actions, &hints, -1),
		NULL, G_DBUS_CALL_FLAGS_NONE, -1, NULL, NULL, NULL);
	g_object_unref(bus);
}

// --------------------------------------------------------------- clipboard

static void desk_clip_text(const char *t) {
	GtkClipboard *c = gtk_clipboard_get(GDK_SELECTION_CLIPBOARD);
	gtk_clipboard_set_text(c, t, -1);
	gtk_clipboard_store(c);
}

static int desk_clip_png(const void *data, int n) {
	GdkPixbufLoader *l = gdk_pixbuf_loader_new_with_type("png", NULL);
	if (!l) return 0;
	gboolean ok = gdk_pixbuf_loader_write(l, data, n, NULL) && gdk_pixbuf_loader_close(l, NULL);
	GdkPixbuf *p = ok ? gdk_pixbuf_loader_get_pixbuf(l) : NULL;
	if (p) {
		GtkClipboard *c = gtk_clipboard_get(GDK_SELECTION_CLIPBOARD);
		gtk_clipboard_set_image(c, p);
		gtk_clipboard_store(c);
	}
	g_object_unref(l);
	return p != NULL;
}

// ------------------------------------------------------------------- icon

static void desk_window_icon(void *w, const char *path) {
	gtk_window_set_icon_from_file(GTK_WINDOW(w), path, NULL);
	gtk_window_set_default_icon_from_file(path, NULL);
}

// -------------------------------------------------------------------- keys

static const SecretSchema *desk_schema(void) {
	static const SecretSchema s = {
		"com.lootnest.Key", SECRET_SCHEMA_NONE,
		{ { "name", SECRET_SCHEMA_ATTRIBUTE_STRING }, { NULL, 0 } }
	};
	return &s;
}

// The key's text, or NULL when there is none; *err is set when the Secret
// Service can't be reached.
static char *desk_key_lookup(const char *name, char **err) {
	GError *e = NULL;
	gchar *v = secret_password_lookup_sync(desk_schema(), NULL, &e, "name", name, NULL);
	if (e) { *err = g_strdup(e->message); g_error_free(e); return NULL; }
	return v;
}

static int desk_key_store(const char *name, const char *label, const char *value, char **err) {
	GError *e = NULL;
	gboolean ok = secret_password_store_sync(desk_schema(), SECRET_COLLECTION_DEFAULT, label, value, NULL, &e, "name", name, NULL);
	if (e) { *err = g_strdup(e->message); g_error_free(e); return 0; }
	return ok;
}

// -------------------------------------------------------------------- tray

// Ayatana AppIndicator (StatusNotifierItem) is loaded when the desktop has
// it, so Lootnest starts without it, with no tray.
typedef void *(*ai_new_fn)(const char *, const char *, int, const char *);
typedef void (*ai_status_fn)(void *, int);
typedef void (*ai_menu_fn)(void *, GtkMenu *);
typedef void (*ai_title_fn)(void *, const char *);
static ai_new_fn ai_new;
static ai_status_fn ai_status;
static ai_menu_fn ai_menu;
static ai_title_fn ai_title;

static int desk_tray_load(void) {
	if (ai_new) return 1;
	const char *libs[] = { "libayatana-appindicator3.so.1", "libappindicator3.so.1", NULL };
	for (int i = 0; libs[i]; i++) {
		void *h = dlopen(libs[i], RTLD_LAZY | RTLD_LOCAL);
		if (!h) continue;
		ai_new = (ai_new_fn)dlsym(h, "app_indicator_new_with_path");
		ai_status = (ai_status_fn)dlsym(h, "app_indicator_set_status");
		ai_menu = (ai_menu_fn)dlsym(h, "app_indicator_set_menu");
		ai_title = (ai_title_fn)dlsym(h, "app_indicator_set_title");
		if (ai_new && ai_status && ai_menu) return 1;
		ai_new = NULL;
	}
	return 0;
}

static void *desk_tray_new(const char *id, const char *icon_dir) {
	if (!desk_tray_load()) return NULL;
	// Category 0 is APPLICATION_STATUS.
	void *t = ai_new(id, "lootnest", 0, icon_dir);
	return t;
}

static void desk_tray_show(void *t, int on, const char *title) {
	if (ai_title && title) ai_title(t, title);
	ai_status(t, on ? 1 : 0); // ACTIVE, PASSIVE
}

static void desk_tray_pick(GtkMenuItem *item, gpointer tray) {
	deskTrayPick((uintptr_t)tray, GPOINTER_TO_INT(g_object_get_data(G_OBJECT(item), "ln-id")));
}

static GtkWidget *desk_menu_new(void) { return gtk_menu_new(); }

static void desk_menu_add(GtkWidget *m, uintptr_t tray, int id, const char *text, int disabled, int sep) {
	GtkWidget *it = sep ? gtk_separator_menu_item_new() : gtk_menu_item_new_with_label(text);
	if (!sep) {
		g_object_set_data(G_OBJECT(it), "ln-id", GINT_TO_POINTER(id));
		g_signal_connect(it, "activate", G_CALLBACK(desk_tray_pick), (gpointer)tray);
		gtk_widget_set_sensitive(it, !disabled);
	}
	gtk_menu_shell_append(GTK_MENU_SHELL(m), it);
}

static void desk_tray_set_menu(void *t, GtkWidget *m) {
	gtk_widget_show_all(m);
	ai_menu(t, GTK_MENU(m));
}
*/
import "C"

import (
	_ "embed"
	"errors"
	"os"
	"path/filepath"
	"runtime/cgo"
	"syscall"
	"unsafe"
)

// IconPNG is Lootnest's icon at 128 px, for the window and the tray.
//
//go:embed icon.png
var IconPNG []byte

var mainTid = syscall.Gettid() // package init runs on the main thread

// OnMain runs f on the GTK thread and waits for it.
func OnMain(f func()) {
	if syscall.Gettid() == mainTid {
		f()
		return
	}
	done := make(chan struct{})
	C.desk_dispatch(C.uintptr_t(cgo.NewHandle(func() { f(); close(done) })))
	<-done
}

//export deskIdle
func deskIdle(fn C.uintptr_t) C.gboolean {
	h := cgo.Handle(fn)
	f := h.Value().(func())
	h.Delete()
	f()
	return C.FALSE
}

// Init starts GTK when no window has (internal/webkit does it for the app):
// false when there's no display.
func Init() bool { return C.desk_init() != 0 }

// Main runs the GTK loop until Quit.
func Main() { C.desk_main() }

// Quit ends Main; any goroutine may call it.
func Quit() { C.desk_quit() }

// ClipboardText reads the clipboard's text.
func ClipboardText() string {
	var out string
	OnMain(func() {
		if c := C.desk_clip_get(); c != nil {
			out = C.GoString(c)
			C.g_free(C.gpointer(c))
		}
	})
	return out
}

// SetParent makes dialogs belong to Lootnest's window (a GtkWindow) and
// gives it Lootnest's icon, written to iconPath.
func SetParent(window unsafe.Pointer, iconPath string) {
	C.desk_set_parent(window)
	if iconPath != "" && os.MkdirAll(filepath.Dir(iconPath), 0o755) == nil && os.WriteFile(iconPath, IconPNG, 0o644) == nil {
		p := C.CString(iconPath)
		defer C.free(unsafe.Pointer(p))
		C.desk_window_icon(window, p)
	}
}

func cstr(s string) *C.char { return C.CString(s) }
func free(p *C.char)        { C.free(unsafe.Pointer(p)) }

// Dialog kinds and button sets for Message.
const (
	Info = iota
	Warning
	Question
	Error
)

const (
	ButtonsOK = iota
	ButtonsOKCancel
	ButtonsYesNo
	ButtonsYesNoCancel
)

// Answers from Message.
const (
	AnswerNone   = 0
	AnswerOK     = 1
	AnswerCancel = 2
	AnswerYes    = 6
	AnswerNo     = 7
)

// Message shows a modal dialog and returns the answer (Windows' numbers:
// AnswerOK, AnswerCancel, AnswerYes, AnswerNo; AnswerNone when closed).
func Message(title, text string, kind, buttons int, defaultNo bool) int {
	var r C.int
	OnMain(func() {
		t, x := cstr(title), cstr(text)
		defer free(t)
		defer free(x)
		dn := C.int(0)
		if defaultNo {
			dn = 1
		}
		r = C.desk_message(t, x, C.int(kind), C.int(buttons), dn)
	})
	switch r {
	case C.GTK_RESPONSE_OK:
		return AnswerOK
	case C.GTK_RESPONSE_CANCEL:
		return AnswerCancel
	case C.GTK_RESPONSE_YES:
		return AnswerYes
	case C.GTK_RESPONSE_NO:
		return AnswerNo
	}
	return AnswerNone
}

// Filter is a named list of ';'-separated patterns ("*.zip;*.7z").
type Filter struct{ Name, Patterns string }

// ErrCancelled is Browse's answer when nothing was chosen.
var ErrCancelled = errors.New("cancelled")

// Browse asks for a file, or a folder, with the desktop's own picker.
func Browse(title string, folder bool, start string, filters []Filter) (string, error) {
	var out string
	OnMain(func() {
		t, s := cstr(title), cstr(start)
		defer free(t)
		defer free(s)
		n := len(filters)
		names := C.malloc(C.size_t(n+1) * C.size_t(unsafe.Sizeof(uintptr(0))))
		specs := C.malloc(C.size_t(n+1) * C.size_t(unsafe.Sizeof(uintptr(0))))
		defer C.free(names)
		defer C.free(specs)
		ns := unsafe.Slice((**C.char)(names), n+1)
		ss := unsafe.Slice((**C.char)(specs), n+1)
		for i, f := range filters {
			ns[i], ss[i] = cstr(f.Name), cstr(f.Patterns)
		}
		defer func() {
			for i := range filters {
				free(ns[i])
				free(ss[i])
			}
		}()
		f := C.int(0)
		if folder {
			f = 1
		}
		if r := C.desk_browse(t, f, s, (**C.char)(names), (**C.char)(specs), C.int(n)); r != nil {
			out = C.GoString(r)
			C.g_free(C.gpointer(r))
		}
	})
	if out == "" {
		return "", ErrCancelled
	}
	return out, nil
}

// Notify shows a desktop notification; iconPath may be "".
func Notify(title, text, iconPath string) {
	t, x, i := cstr(title), cstr(text), cstr(iconPath)
	defer free(t)
	defer free(x)
	defer free(i)
	C.desk_notify(t, x, i)
}

// SetClipboardText puts text on the clipboard.
func SetClipboardText(text string) {
	OnMain(func() {
		t := cstr(text)
		defer free(t)
		C.desk_clip_text(t)
	})
}

// SetClipboardPNG puts an image on the clipboard.
func SetClipboardPNG(png []byte) error {
	if len(png) == 0 {
		return errors.New("no image")
	}
	var ok C.int
	OnMain(func() {
		p := C.CBytes(png)
		defer C.free(p)
		ok = C.desk_clip_png(p, C.int(len(png)))
	})
	if ok == 0 {
		return errors.New("the image couldn't be read")
	}
	return nil
}

// KeyLookup reads a key from the Secret Service: "" with a nil error when
// there is none; an error when the Secret Service can't be reached.
func KeyLookup(name string) (string, error) {
	n := cstr(name)
	defer free(n)
	var cerr *C.char
	v := C.desk_key_lookup(n, &cerr)
	if cerr != nil {
		defer C.g_free(C.gpointer(cerr))
		return "", errors.New(C.GoString(cerr))
	}
	if v == nil {
		return "", nil
	}
	defer C.secret_password_free(v)
	return C.GoString(v), nil
}

// KeyStore keeps a key in the Secret Service's default keyring.
func KeyStore(name, label, value string) error {
	n, l, v := cstr(name), cstr(label), cstr(value)
	defer free(n)
	defer free(l)
	defer free(v)
	var cerr *C.char
	if C.desk_key_store(n, l, v, &cerr) == 0 {
		if cerr != nil {
			defer C.g_free(C.gpointer(cerr))
			return errors.New(C.GoString(cerr))
		}
		return errors.New("the keyring didn't keep the key")
	}
	return nil
}

// MenuItem is one line of the tray's menu.
type MenuItem struct {
	ID       int
	Text     string
	Disabled bool
	Sep      bool
}

// Tray is Lootnest's icon in the desktop's tray (StatusNotifierItem through
// Ayatana AppIndicator). Its menu is all there is: a click opens it.
type Tray struct {
	ind    unsafe.Pointer
	handle cgo.Handle
	pick   func(id int)
}

// NewTray makes the icon, or returns nil when the desktop has no
// AppIndicator library. The icon is written under iconDir.
func NewTray(iconDir string) *Tray {
	if os.MkdirAll(iconDir, 0o755) != nil || os.WriteFile(filepath.Join(iconDir, "lootnest.png"), IconPNG, 0o644) != nil {
		return nil
	}
	var t *Tray
	OnMain(func() {
		id, dir := cstr("lootnest"), cstr(iconDir)
		defer free(id)
		defer free(dir)
		if ind := C.desk_tray_new(id, dir); ind != nil {
			t = &Tray{ind: ind}
			t.handle = cgo.NewHandle(t)
		}
	})
	return t
}

// Show shows or hides the icon.
func (t *Tray) Show(on bool, title string) {
	if t == nil {
		return
	}
	OnMain(func() {
		c := cstr(title)
		defer free(c)
		v := C.int(0)
		if on {
			v = 1
		}
		C.desk_tray_show(t.ind, v, c)
	})
}

// SetMenu replaces the menu; pick is told the ID of the line chosen, on the
// GTK thread.
func (t *Tray) SetMenu(items []MenuItem, pick func(id int)) {
	if t == nil {
		return
	}
	OnMain(func() {
		t.pick = pick
		m := C.desk_menu_new()
		for _, it := range items {
			c := cstr(it.Text)
			d, s := C.int(0), C.int(0)
			if it.Disabled {
				d = 1
			}
			if it.Sep {
				s = 1
			}
			C.desk_menu_add(m, C.uintptr_t(t.handle), C.int(it.ID), c, d, s)
			free(c)
		}
		C.desk_tray_set_menu(t.ind, m)
	})
}

//export deskTrayPick
func deskTrayPick(tray C.uintptr_t, id C.int) {
	t := cgo.Handle(tray).Value().(*Tray)
	if t.pick != nil {
		t.pick(int(id))
	}
}
