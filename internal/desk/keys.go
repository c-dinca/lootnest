//go:build linux

package desk

/*
#cgo pkg-config: gtk+-3.0 x11
#include <stdlib.h>
#include <stdint.h>
#include <string.h>
#include <unistd.h>
#include <gtk/gtk.h>
#include <gdk/gdkx.h>
#include <X11/Xlib.h>
#include <X11/Xatom.h>

extern void deskKey(unsigned int keycode, unsigned int state);
extern void deskPortal(uintptr_t call, int code, char *text);
extern void deskShortcut(char *id);

// ------------------------------------------------------- X11 key grabs

static int desk_x11(void) {
	GdkDisplay *d = gdk_display_get_default();
	return d && GDK_IS_X11_DISPLAY(d);
}

static GdkFilterReturn desk_key_filter(GdkXEvent *gx, GdkEvent *e, gpointer data) {
	XEvent *x = (XEvent *)gx;
	if (x->type == KeyPress) deskKey(x->xkey.keycode, x->xkey.state & (ShiftMask | ControlMask | Mod1Mask | Mod4Mask));
	return GDK_FILTER_CONTINUE;
}

// desk_grab holds (on) or lets go of a key on the root window, with Caps
// Lock and Num Lock either way. It answers the key's code, or 0 when there
// is none or another program holds it.
static unsigned int desk_grab(unsigned long keysym, unsigned int mods, int on) {
	static int filtered = 0;
	GdkDisplay *d = gdk_display_get_default();
	Display *x = GDK_DISPLAY_XDISPLAY(d);
	KeyCode kc = XKeysymToKeycode(x, keysym);
	if (!kc) return 0;
	if (!filtered) {
		gdk_window_add_filter(NULL, desk_key_filter, NULL);
		filtered = 1;
	}
	unsigned int extra[] = {0, LockMask, Mod2Mask, LockMask | Mod2Mask};
	Window root = DefaultRootWindow(x);
	gdk_x11_display_error_trap_push(d);
	for (int i = 0; i < 4; i++) {
		if (on) XGrabKey(x, kc, mods | extra[i], root, False, GrabModeAsync, GrabModeAsync);
		else XUngrabKey(x, kc, mods | extra[i], root);
	}
	XSync(x, False);
	if (gdk_x11_display_error_trap_pop(d) != 0) {
		if (on) desk_grab(keysym, mods, 0);
		return 0;
	}
	return kc;
}

// ------------------------------------------------------ X11 windows

// desk_windows lists the desktop's windows bottom to top as
// (xid, pid, x, y, w, h) sextets, viewable ones only; it answers how many.
static int desk_windows(long *out, int max) {
	Display *x = GDK_DISPLAY_XDISPLAY(gdk_display_get_default());
	Window root = DefaultRootWindow(x);
	Atom list = XInternAtom(x, "_NET_CLIENT_LIST_STACKING", False), pidA = XInternAtom(x, "_NET_WM_PID", False);
	Atom type; int format; unsigned long n = 0, after; unsigned char *data = NULL;
	if (XGetWindowProperty(x, root, list, 0, 4096, False, XA_WINDOW, &type, &format, &n, &after, &data) != Success || !data) return 0;
	int k = 0;
	Window *wins = (Window *)data;
	gdk_x11_display_error_trap_push(gdk_display_get_default());
	for (unsigned long i = 0; i < n && k < max; i++) {
		XWindowAttributes at;
		if (!XGetWindowAttributes(x, wins[i], &at) || at.map_state != IsViewable) continue;
		long pid = 0;
		unsigned char *p = NULL; unsigned long pn = 0;
		if (XGetWindowProperty(x, wins[i], pidA, 0, 1, False, XA_CARDINAL, &type, &format, &pn, &after, &p) == Success && p) {
			if (pn) pid = (long)*(unsigned long *)p;
			XFree(p);
		}
		int rx = 0, ry = 0; Window child;
		XTranslateCoordinates(x, wins[i], root, 0, 0, &rx, &ry, &child);
		long *o = out + 6 * k++;
		o[0] = (long)wins[i]; o[1] = pid; o[2] = rx; o[3] = ry; o[4] = at.width; o[5] = at.height;
	}
	gdk_x11_display_error_trap_pop_ignored(gdk_display_get_default());
	XFree(data);
	return k;
}

// desk_capture copies a window's picture (screen: the screen where it is
// instead) into a new RGBA buffer; NULL when it can't.
static unsigned char *desk_capture(unsigned long xid, int screen, int *w, int *h) {
	GdkDisplay *d = gdk_display_get_default();
	GdkWindow *gw = gdk_x11_window_foreign_new_for_display(d, xid);
	if (!gw) return NULL;
	int x = 0, y = 0;
	*w = gdk_window_get_width(gw);
	*h = gdk_window_get_height(gw);
	GdkPixbuf *pb;
	if (screen) {
		gdk_window_get_origin(gw, &x, &y);
		pb = gdk_pixbuf_get_from_window(gdk_get_default_root_window(), x, y, *w, *h);
	} else {
		pb = gdk_pixbuf_get_from_window(gw, 0, 0, *w, *h);
	}
	g_object_unref(gw);
	if (!pb) return NULL;
	*w = gdk_pixbuf_get_width(pb);
	*h = gdk_pixbuf_get_height(pb);
	int stride = gdk_pixbuf_get_rowstride(pb), nc = gdk_pixbuf_get_n_channels(pb);
	const guchar *src = gdk_pixbuf_read_pixels(pb);
	unsigned char *out = malloc((size_t)(*w) * (*h) * 4);
	for (int r = 0; r < *h; r++) for (int c = 0; c < *w; c++) {
		const guchar *s = src + r * stride + c * nc;
		unsigned char *o = out + ((size_t)r * (*w) + c) * 4;
		o[0] = s[0]; o[1] = s[1]; o[2] = s[2]; o[3] = 255;
	}
	g_object_unref(pb);
	return out;
}

// ------------------------------------------- the desktop portal (D-Bus)

#define PORTAL "org.freedesktop.portal.Desktop"
#define PORTAL_PATH "/org/freedesktop/portal/desktop"

// The session bus, on first use telling the portal Lootnest's app id
// (xdg-desktop-portal 1.19 and newer; older ones take it from the desktop
// entry it was started from, and the error is ignored).
static GDBusConnection *desk_bus(void) {
	static GDBusConnection *bus = NULL;
	if (bus) return bus;
	bus = g_bus_get_sync(G_BUS_TYPE_SESSION, NULL, NULL);
	if (!bus) return NULL;
	GVariant *r = g_dbus_connection_call_sync(bus, PORTAL, PORTAL_PATH, "org.freedesktop.host.portal.Registry", "Register",
		g_variant_new("(sa{sv})", "lootnest", NULL), NULL, G_DBUS_CALL_FLAGS_NONE, 2000, NULL, NULL);
	if (r) g_variant_unref(r);
	return bus;
}

// A portal call answers with a Request object whose Response signal
// carries the result; its path is known before the call from the token.
typedef struct { uintptr_t call; guint sub; int kind; } desk_req;

static char *desk_req_path(const char *token) {
	gchar *s = g_strdup(g_dbus_connection_get_unique_name(desk_bus()) + 1);
	for (gchar *p = s; *p; p++) if (*p == '.') *p = '_';
	char *path = g_strdup_printf(PORTAL_PATH "/request/%s/%s", s, token);
	g_free(s);
	return path;
}

static char *desk_token(void) {
	static int n = 0;
	return g_strdup_printf("lootnest%d_%d", (int)getpid(), ++n);
}

// The Response's results as text for Go: for a screenshot its uri, for a
// session its handle, for shortcuts "id=trigger" lines.
static char *desk_results(GVariant *res) {
	GVariantIter it; const char *k; GVariant *v;
	GString *out = g_string_new("");
	g_variant_iter_init(&it, res);
	while (g_variant_iter_next(&it, "{&sv}", &k, &v)) {
		if ((!strcmp(k, "uri") || !strcmp(k, "session_handle")) &&
			(g_variant_is_of_type(v, G_VARIANT_TYPE_STRING) || g_variant_is_of_type(v, G_VARIANT_TYPE_OBJECT_PATH))) {
			g_string_append(out, g_variant_get_string(v, NULL));
		} else if (!strcmp(k, "shortcuts") && g_variant_is_of_type(v, G_VARIANT_TYPE("a(sa{sv})"))) {
			GVariantIter si; const char *id; GVariant *props;
			g_variant_iter_init(&si, v);
			while (g_variant_iter_next(&si, "(&s@a{sv})", &id, &props)) {
				const char *trig = "";
				g_variant_lookup(props, "trigger_description", "&s", &trig);
				g_string_append_printf(out, "%s=%s\n", id, trig);
				g_variant_unref(props);
			}
		}
		g_variant_unref(v);
	}
	return g_string_free(out, FALSE);
}

static void desk_on_response(GDBusConnection *c, const gchar *sender, const gchar *path, const gchar *iface,
	const gchar *signal, GVariant *params, gpointer data) {
	desk_req *r = data;
	guint32 code = 2; GVariant *res = NULL;
	g_variant_get(params, "(u@a{sv})", &code, &res);
	char *text = res ? desk_results(res) : g_strdup("");
	if (res) g_variant_unref(res);
	g_dbus_connection_signal_unsubscribe(c, r->sub);
	deskPortal(r->call, (int)code, text);
	g_free(text);
	g_free(r);
}

static void desk_on_called(GObject *src, GAsyncResult *res, gpointer data) {
	desk_req *r = data;
	GError *err = NULL;
	GVariant *v = g_dbus_connection_call_finish(G_DBUS_CONNECTION(src), res, &err);
	if (v) { g_variant_unref(v); return; } // the Response follows
	g_dbus_connection_signal_unsubscribe(desk_bus(), r->sub);
	deskPortal(r->call, -1, err ? err->message : "the portal didn't answer");
	if (err) g_error_free(err);
	g_free(r);
}

// desk_portal_call calls method on a portal interface with params whose
// last member is the options dictionary, to which it adds handle_token.
// The result reaches Go as deskPortal(call, response code, text); -1 is
// a call that failed (no portal, or no such interface).
static int desk_portal_call(uintptr_t call, const char *iface, const char *method, GVariant *params) {
	GDBusConnection *bus = desk_bus();
	if (!bus) { g_variant_unref(g_variant_ref_sink(params)); return 0; }
	char *token = desk_token(), *path = desk_req_path(token);
	// Put the token into the options (the tuple's last member).
	gsize n = g_variant_n_children(params);
	GVariant **kids = g_new(GVariant *, n);
	for (gsize i = 0; i < n; i++) kids[i] = g_variant_get_child_value(params, i);
	GVariantBuilder opts;
	g_variant_builder_init(&opts, G_VARIANT_TYPE("a{sv}"));
	GVariantIter it; const char *k; GVariant *v;
	g_variant_iter_init(&it, kids[n - 1]);
	while (g_variant_iter_next(&it, "{&sv}", &k, &v)) { g_variant_builder_add(&opts, "{sv}", k, v); g_variant_unref(v); }
	g_variant_builder_add(&opts, "{sv}", "handle_token", g_variant_new_string(token));
	g_variant_unref(kids[n - 1]);
	kids[n - 1] = g_variant_builder_end(&opts);
	GVariant *full = g_variant_new_tuple(kids, n);
	g_free(kids);
	g_variant_unref(g_variant_ref_sink(params));

	desk_req *r = g_new0(desk_req, 1);
	r->call = call;
	r->sub = g_dbus_connection_signal_subscribe(bus, PORTAL, "org.freedesktop.portal.Request", "Response", path, NULL,
		G_DBUS_SIGNAL_FLAGS_NONE, desk_on_response, r, NULL);
	g_dbus_connection_call(bus, PORTAL, PORTAL_PATH, iface, method, full, NULL, G_DBUS_CALL_FLAGS_NONE, 30000, NULL, desk_on_called, r);
	g_free(token);
	g_free(path);
	return 1;
}

static int desk_screenshot(uintptr_t call) {
	return desk_portal_call(call, "org.freedesktop.portal.Screenshot", "Screenshot",
		g_variant_new("(s@a{sv})", "", g_variant_new_parsed("{'interactive': <false>, 'modal': <false>}")));
}

static int desk_shortcuts_session(uintptr_t call) {
	char *tok = desk_token();
	GVariantBuilder o;
	g_variant_builder_init(&o, G_VARIANT_TYPE("a{sv}"));
	g_variant_builder_add(&o, "{sv}", "session_handle_token", g_variant_new_string(tok));
	g_free(tok);
	return desk_portal_call(call, "org.freedesktop.portal.GlobalShortcuts", "CreateSession", g_variant_new("(a{sv})", &o));
}

// desk_shortcuts_bind binds "id\tdescription\ttrigger" lines.
static int desk_shortcuts_bind(uintptr_t call, const char *session, const char *lines) {
	GVariantBuilder list;
	g_variant_builder_init(&list, G_VARIANT_TYPE("a(sa{sv})"));
	gchar **ls = g_strsplit(lines, "\n", -1);
	for (gchar **l = ls; *l; l++) {
		gchar **f = g_strsplit(*l, "\t", 3);
		if (f[0] && f[1] && f[2]) {
			GVariantBuilder p;
			g_variant_builder_init(&p, G_VARIANT_TYPE("a{sv}"));
			g_variant_builder_add(&p, "{sv}", "description", g_variant_new_string(f[1]));
			g_variant_builder_add(&p, "{sv}", "preferred_trigger", g_variant_new_string(f[2]));
			g_variant_builder_add(&list, "(sa{sv})", f[0], &p);
		}
		g_strfreev(f);
	}
	g_strfreev(ls);
	return desk_portal_call(call, "org.freedesktop.portal.GlobalShortcuts", "BindShortcuts",
		g_variant_new("(oa(sa{sv})sa{sv})", session, &list, "", NULL));
}

static void desk_on_activated(GDBusConnection *c, const gchar *sender, const gchar *path, const gchar *iface,
	const gchar *signal, GVariant *params, gpointer data) {
	const char *session, *id;
	g_variant_get(params, "(&o&sta{sv})", &session, &id, NULL, NULL);
	if (!strcmp(session, (const char *)data)) deskShortcut((char *)id);
}

static void desk_shortcuts_listen(const char *session) {
	g_dbus_connection_signal_subscribe(desk_bus(), PORTAL, "org.freedesktop.portal.GlobalShortcuts", "Activated",
		PORTAL_PATH, NULL, G_DBUS_SIGNAL_FLAGS_NONE, desk_on_activated, g_strdup(session), g_free);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"image"
	"net/url"
	"runtime/cgo"
	"strings"
	"sync"
	"time"
	"unsafe"
)

// X11 tells whether the window's display is X11, where Lootnest can hold
// keys and see other programs' windows itself. On Wayland it asks the
// desktop portal instead.
func X11() bool {
	var on bool
	OnMain(func() { on = C.desk_x11() != 0 })
	return on
}

// X11 key masks, for GrabKey.
const (
	ShiftMask   = C.ShiftMask
	ControlMask = C.ControlMask
	AltMask     = C.Mod1Mask
)

var (
	keysMu  sync.Mutex
	grabbed = map[uint64]func(){} // key code and mods, to what it runs
)

// GrabKey holds a key with mods for the whole desktop on X11, running
// fired on the GTK thread when it's pressed. False when another program
// holds it.
func GrabKey(keysym, mods uint32, fired func()) bool {
	var kc C.uint
	OnMain(func() { kc = C.desk_grab(C.ulong(keysym), C.uint(mods), 1) })
	if kc == 0 {
		return false
	}
	keysMu.Lock()
	grabbed[uint64(kc)<<32|uint64(mods)] = fired
	keysMu.Unlock()
	return true
}

// UngrabKey lets go of a key GrabKey held.
func UngrabKey(keysym, mods uint32) {
	var kc C.uint
	OnMain(func() { kc = C.desk_grab(C.ulong(keysym), C.uint(mods), 0) })
	keysMu.Lock()
	for k := range grabbed {
		if k&0xffffffff == uint64(mods) && (kc == 0 || k>>32 == uint64(kc)) {
			delete(grabbed, k)
		}
	}
	keysMu.Unlock()
}

//export deskKey
func deskKey(keycode, state C.uint) {
	keysMu.Lock()
	f := grabbed[uint64(keycode)<<32|uint64(state)]
	keysMu.Unlock()
	if f != nil {
		f()
	}
}

// XWindow is another program's window on X11.
type XWindow struct {
	ID         uintptr
	PID        int
	X, Y, W, H int
}

// X11Windows lists the desktop's shown windows, bottom to top; none on
// Wayland.
func X11Windows() []XWindow {
	var out []XWindow
	OnMain(func() {
		if C.desk_x11() == 0 {
			return
		}
		buf := make([]C.long, 6*512)
		n := int(C.desk_windows(&buf[0], 512))
		for i := 0; i < n; i++ {
			o := buf[6*i:]
			out = append(out, XWindow{ID: uintptr(o[0]), PID: int(o[1]), X: int(o[2]), Y: int(o[3]), W: int(o[4]), H: int(o[5])})
		}
	})
	return out
}

// X11Capture copies a window's picture; fromScreen takes what the screen
// shows where it is instead.
func X11Capture(id uintptr, fromScreen bool) (*image.RGBA, error) {
	var img *image.RGBA
	OnMain(func() {
		if C.desk_x11() == 0 {
			return
		}
		var w, h C.int
		scr := C.int(0)
		if fromScreen {
			scr = 1
		}
		p := C.desk_capture(C.ulong(id), scr, &w, &h)
		if p == nil {
			return
		}
		defer C.free(unsafe.Pointer(p))
		img = image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
		copy(img.Pix, unsafe.Slice((*byte)(unsafe.Pointer(p)), int(w)*int(h)*4))
	})
	if img == nil {
		return nil, errors.New("couldn't copy the window's picture")
	}
	return img, nil
}

// portalAnswer is what a portal request ended with.
type portalAnswer struct {
	code int // 0 done, 1 cancelled by the user, 2 other; -1 the call failed
	text string
}

//export deskPortal
func deskPortal(call C.uintptr_t, code C.int, text *C.char) {
	h := cgo.Handle(call)
	f := h.Value().(func(portalAnswer))
	h.Delete()
	f(portalAnswer{int(code), C.GoString(text)})
}

// portal starts a portal call on the GTK thread; done gets its answer
// there.
func portal(start func(call C.uintptr_t) C.int, done func(portalAnswer)) {
	OnMain(func() {
		h := cgo.NewHandle(done)
		if start(C.uintptr_t(h)) == 0 {
			h.Delete()
			done(portalAnswer{-1, "no session bus"})
		}
	})
}

func (a portalAnswer) err(what string) error {
	switch a.code {
	case 0:
		return nil
	case 1:
		return fmt.Errorf("%s: turned down", what)
	case -1:
		return fmt.Errorf("%s: the desktop has no portal for it (%s)", what, a.text)
	}
	return fmt.Errorf("%s: the portal said no", what)
}

// Screenshot asks the desktop portal for a picture of the screen, without
// asking the user each time where the desktop allows it, and answers the
// file it saved. Don't call it on the GTK thread: it waits for the portal.
func Screenshot() (string, error) {
	got := make(chan portalAnswer, 1)
	portal(func(call C.uintptr_t) C.int { return C.desk_screenshot(call) }, func(a portalAnswer) { got <- a })
	select {
	case a := <-got:
		if err := a.err("Screenshot"); err != nil {
			return "", err
		}
		u, err := url.Parse(strings.TrimSpace(a.text))
		if err != nil || u.Scheme != "file" || u.Path == "" {
			return "", fmt.Errorf("Screenshot: the portal gave %q", a.text)
		}
		return u.Path, nil
	case <-time.After(20 * time.Second):
		return "", errors.New("Screenshot: the portal didn't answer")
	}
}

// Shortcut is a key for the whole desktop, asked of the GlobalShortcuts
// portal.
type Shortcut struct {
	ID      string
	Says    string // what it does, shown by the desktop
	Trigger string // the key Lootnest asks for, in the portal's spelling ("CTRL+SHIFT+F5")
}

//export deskShortcut
func deskShortcut(id *C.char) {
	keysMu.Lock()
	f := shortcutFired
	keysMu.Unlock()
	if f != nil {
		f(C.GoString(id))
	}
}

var shortcutFired func(id string)

// BindShortcuts asks the GlobalShortcuts portal for keys: a session, then
// the keys in it (the desktop may ask the user once, and may give other
// keys than the ones asked for). fired runs on the GTK thread for each
// press; done gets the keys given ("id" to how the desktop names its key)
// or why there are none.
func BindShortcuts(keys []Shortcut, fired func(id string), done func(got map[string]string, err error)) {
	keysMu.Lock()
	shortcutFired = fired
	keysMu.Unlock()
	var lines []string
	for _, k := range keys {
		clean := strings.NewReplacer("\t", " ", "\n", " ")
		lines = append(lines, clean.Replace(k.ID)+"\t"+clean.Replace(k.Says)+"\t"+clean.Replace(k.Trigger))
	}
	portal(func(call C.uintptr_t) C.int { return C.desk_shortcuts_session(call) }, func(a portalAnswer) {
		if err := a.err("Keys"); err != nil {
			done(nil, err)
			return
		}
		session := strings.TrimSpace(a.text)
		if session == "" {
			done(nil, errors.New("Keys: the portal gave no session"))
			return
		}
		cs := C.CString(session)
		C.desk_shortcuts_listen(cs)
		cl := C.CString(strings.Join(lines, "\n"))
		defer C.free(unsafe.Pointer(cs))
		defer C.free(unsafe.Pointer(cl))
		h := cgo.NewHandle(func(b portalAnswer) {
			if err := b.err("Keys"); err != nil {
				done(nil, err)
				return
			}
			got := map[string]string{}
			for _, l := range strings.Split(b.text, "\n") {
				if id, trig, ok := strings.Cut(l, "="); ok {
					got[id] = trig
				}
			}
			done(got, nil)
		})
		if C.desk_shortcuts_bind(C.uintptr_t(h), cs, cl) == 0 {
			h.Delete()
			done(nil, errors.New("Keys: no session bus"))
		}
	})
}

// cUint is C's unsigned int, for the tests (cgo isn't allowed in them).
type cUint = C.uint
