//go:build linux

// Package webkit is Lootnest's window on Linux: a GTK 3 window holding a
// WebKitGTK 4.1 view, with the same guarantees as the WebView2 window on
// Windows (third_party/go-webview2):
//
//   - the page is Lootnest's own, loaded from a string under a base address
//     of its own scheme (lootnest://app/), marked secure so the Gamepad API
//     and the clipboard work there;
//   - every navigation away from it and every new window is refused, and so
//     is every permission a page could ask for;
//   - a call from the page reaches Go only through the one message handler,
//     and only when the view still shows that address (AllowMessage);
//   - bound Go functions are called with JSON arguments and answer through a
//     promise, as webview's Bind does.
//
// Everything here runs on the GTK thread, the process's main thread: lock
// it in an init function (runtime.LockOSThread) and call New and Run from
// main. Dispatch is the way in from other goroutines.
package webkit

/*
#cgo pkg-config: gtk+-3.0 webkit2gtk-4.1
#include <stdlib.h>
#include <stdint.h>
#include <gtk/gtk.h>
#include <webkit2/webkit2.h>

extern void lnMessage(uintptr_t view, char *msg, char *uri);
extern gboolean lnClose(uintptr_t view);
extern gboolean lnIdle(uintptr_t fn);
extern void lnState(uintptr_t view, int visible, int active, int full, int max);

static void ln_on_message(WebKitUserContentManager *m, WebKitJavascriptResult *r, gpointer data) {
	JSCValue *v = webkit_javascript_result_get_js_value(r);
	if (!jsc_value_is_string(v)) return;
	char *s = jsc_value_to_string(v);
	// The address of the document last committed in the view: a refused
	// navigation leaves webkit_web_view_get_uri on the address it refused.
	const char *uri = g_object_get_data(G_OBJECT(m), "ln-committed");
	lnMessage((uintptr_t)data, s, (char *)(uri ? uri : ""));
	g_free(s);
}

static void ln_on_load(WebKitWebView *v, WebKitLoadEvent e, gpointer ucm) {
	if (e == WEBKIT_LOAD_COMMITTED) {
		const char *uri = webkit_web_view_get_uri(v);
		g_object_set_data_full(G_OBJECT(ucm), "ln-committed", g_strdup(uri ? uri : ""), g_free);
	}
}

// Only the page's own address may load; nothing opens a new window.
static gboolean ln_on_policy(WebKitWebView *v, WebKitPolicyDecision *d, WebKitPolicyDecisionType t, gpointer base) {
	if (t == WEBKIT_POLICY_DECISION_TYPE_NAVIGATION_ACTION) {
		WebKitNavigationAction *a = webkit_navigation_policy_decision_get_navigation_action(WEBKIT_NAVIGATION_POLICY_DECISION(d));
		const char *uri = webkit_uri_request_get_uri(webkit_navigation_action_get_request(a));
		if (uri && strcmp(uri, (const char *)base) == 0) {
			webkit_policy_decision_use(d);
		} else {
			webkit_policy_decision_ignore(d);
		}
		return TRUE;
	}
	if (t == WEBKIT_POLICY_DECISION_TYPE_NEW_WINDOW_ACTION) {
		webkit_policy_decision_ignore(d);
		return TRUE;
	}
	return FALSE;
}

static gboolean ln_on_permission(WebKitWebView *v, WebKitPermissionRequest *r, gpointer data) {
	if (WEBKIT_IS_CLIPBOARD_PERMISSION_REQUEST(r)) {
		webkit_permission_request_allow(r); // the page's own Copy keys
	} else {
		webkit_permission_request_deny(r);
	}
	return TRUE;
}

static gboolean ln_on_context_menu(WebKitWebView *v, WebKitContextMenu *m, GdkEvent *e, WebKitHitTestResult *h, gpointer data) {
	return TRUE; // the page draws its own menus
}

static gboolean ln_on_delete(GtkWidget *w, GdkEvent *e, gpointer data) {
	return lnClose((uintptr_t)data);
}

// Lootnest's scheme answers nothing: the page is loaded from a string, and
// a request for anything under it fails.
static void ln_on_scheme(WebKitURISchemeRequest *r, gpointer data) {
	GError *err = g_error_new_literal(G_IO_ERROR, G_IO_ERROR_NOT_FOUND, "not found");
	webkit_uri_scheme_request_finish_error(r, err);
	g_error_free(err);
}

// The window's state, kept in Go so any goroutine can read it.
static void ln_note(GtkWidget *w, uintptr_t id) {
	GdkWindow *gw = gtk_widget_get_window(w);
	GdkWindowState st = gw ? gdk_window_get_state(gw) : 0;
	lnState(id, gtk_widget_get_visible(w), gtk_window_is_active(GTK_WINDOW(w)),
		(st & GDK_WINDOW_STATE_FULLSCREEN) != 0, (st & GDK_WINDOW_STATE_MAXIMIZED) != 0);
}
static gboolean ln_on_state(GtkWidget *w, GdkEventWindowState *e, gpointer id) { ln_note(w, (uintptr_t)id); return FALSE; }
static void ln_on_shown(GtkWidget *w, gpointer id) { ln_note(w, (uintptr_t)id); }
static void ln_on_active(GObject *o, GParamSpec *p, gpointer id) { ln_note(GTK_WIDGET(o), (uintptr_t)id); }

static gboolean ln_idle(gpointer fn) { return lnIdle((uintptr_t)fn); }
static void ln_dispatch(uintptr_t fn) { g_idle_add(ln_idle, (gpointer)fn); }

typedef struct {
	GtkWidget *window;
	WebKitWebView *view;
	WebKitUserContentManager *ucm;
} ln_window;

static ln_window *ln_new(uintptr_t id, const char *title, int w, int h, int minw, int minh,
	const char *data_dir, const char *cache_dir, const char *base, double r, double g, double b, int debug) {
	if (!gtk_init_check(NULL, NULL)) return NULL;
	ln_window *lw = g_new0(ln_window, 1);

	WebKitWebsiteDataManager *dm = webkit_website_data_manager_new(
		"base-data-directory", data_dir, "base-cache-directory", cache_dir, NULL);
	WebKitWebContext *ctx = webkit_web_context_new_with_website_data_manager(dm);
	webkit_web_context_register_uri_scheme(ctx, "lootnest", ln_on_scheme, NULL, NULL);
	WebKitSecurityManager *sm = webkit_web_context_get_security_manager(ctx);
	webkit_security_manager_register_uri_scheme_as_secure(sm, "lootnest");
	webkit_security_manager_register_uri_scheme_as_cors_enabled(sm, "lootnest");

	lw->ucm = webkit_user_content_manager_new();
	lw->view = WEBKIT_WEB_VIEW(g_object_new(WEBKIT_TYPE_WEB_VIEW,
		"web-context", ctx, "user-content-manager", lw->ucm, NULL));
	g_signal_connect(lw->view, "load-changed", G_CALLBACK(ln_on_load), lw->ucm);
	g_signal_connect(lw->ucm, "script-message-received::lootnest", G_CALLBACK(ln_on_message), (gpointer)id);
	webkit_user_content_manager_register_script_message_handler(lw->ucm, "lootnest");
	g_signal_connect(lw->view, "decide-policy", G_CALLBACK(ln_on_policy), (gpointer)g_strdup(base));
	g_signal_connect(lw->view, "permission-request", G_CALLBACK(ln_on_permission), NULL);
	if (!debug) g_signal_connect(lw->view, "context-menu", G_CALLBACK(ln_on_context_menu), NULL);

	WebKitSettings *s = webkit_web_view_get_settings(lw->view);
	webkit_settings_set_enable_developer_extras(s, debug ? TRUE : FALSE);
	webkit_settings_set_enable_write_console_messages_to_stdout(s, debug ? TRUE : FALSE);
	webkit_settings_set_javascript_can_open_windows_automatically(s, FALSE);
	webkit_settings_set_enable_back_forward_navigation_gestures(s, FALSE);
	webkit_settings_set_allow_file_access_from_file_urls(s, FALSE);
	webkit_settings_set_allow_universal_access_from_file_urls(s, FALSE);

	GdkRGBA bg = {r, g, b, 1.0};
	webkit_web_view_set_background_color(lw->view, &bg);

	lw->window = gtk_window_new(GTK_WINDOW_TOPLEVEL);
	gtk_window_set_title(GTK_WINDOW(lw->window), title);
	gtk_window_set_default_size(GTK_WINDOW(lw->window), w, h);
	gtk_window_set_position(GTK_WINDOW(lw->window), GTK_WIN_POS_CENTER);
	if (minw > 0 && minh > 0) {
		GdkGeometry geo = {0};
		geo.min_width = minw;
		geo.min_height = minh;
		gtk_window_set_geometry_hints(GTK_WINDOW(lw->window), NULL, &geo, GDK_HINT_MIN_SIZE);
	}
	gtk_container_add(GTK_CONTAINER(lw->window), GTK_WIDGET(lw->view));
	g_signal_connect(lw->window, "delete-event", G_CALLBACK(ln_on_delete), (gpointer)id);
	g_signal_connect(lw->window, "window-state-event", G_CALLBACK(ln_on_state), (gpointer)id);
	g_signal_connect(lw->window, "show", G_CALLBACK(ln_on_shown), (gpointer)id);
	g_signal_connect(lw->window, "hide", G_CALLBACK(ln_on_shown), (gpointer)id);
	g_signal_connect(lw->window, "notify::is-active", G_CALLBACK(ln_on_active), (gpointer)id);
	gtk_widget_grab_focus(GTK_WIDGET(lw->view));
	return lw;
}

static void ln_init_script(ln_window *lw, const char *js) {
	WebKitUserScript *s = webkit_user_script_new(js, WEBKIT_USER_CONTENT_INJECT_TOP_FRAME,
		WEBKIT_USER_SCRIPT_INJECT_AT_DOCUMENT_START, NULL, NULL);
	webkit_user_content_manager_add_script(lw->ucm, s);
	webkit_user_script_unref(s);
}

static void ln_eval(ln_window *lw, const char *js) {
	webkit_web_view_evaluate_javascript(lw->view, js, -1, NULL, NULL, NULL, NULL, NULL);
}
static void ln_load_html(ln_window *lw, const char *html, const char *base) { webkit_web_view_load_html(lw->view, html, base); }
static void ln_show(ln_window *lw) { gtk_widget_show_all(lw->window); }
static void ln_present(ln_window *lw) { gtk_widget_show_all(lw->window); gtk_window_present(GTK_WINDOW(lw->window)); }
static void ln_hide(ln_window *lw) { gtk_widget_hide(lw->window); }
static void ln_set_title(ln_window *lw, const char *t) { gtk_window_set_title(GTK_WINDOW(lw->window), t); }
static void ln_resize(ln_window *lw, int w, int h) { gtk_window_resize(GTK_WINDOW(lw->window), w, h); }
static void ln_fullscreen(ln_window *lw, int on) {
	if (on) gtk_window_fullscreen(GTK_WINDOW(lw->window)); else gtk_window_unfullscreen(GTK_WINDOW(lw->window));
}
// Full screen on the display's primary monitor (the first when none is).
static void ln_fullscreen_primary(ln_window *lw) {
	GdkDisplay *d = gtk_widget_get_display(lw->window);
	GdkMonitor *p = gdk_display_get_primary_monitor(d);
	int n = gdk_display_get_n_monitors(d), at = 0;
	for (int i = 0; i < n; i++) if (gdk_display_get_monitor(d, i) == p) at = i;
	gtk_window_fullscreen_on_monitor(GTK_WINDOW(lw->window), gtk_widget_get_screen(lw->window), at);
}
static void ln_destroy(ln_window *lw) { gtk_widget_destroy(lw->window); }
static void ln_quit(void) { gtk_main_quit(); }
static void ln_run(void) { gtk_main(); }
*/
import "C"

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"reflect"
	"runtime/cgo"
	"strconv"
	"sync"
	"sync/atomic"
	"unsafe"
)

// BaseURI is the page's address: its own scheme, so nothing it asks for
// under it is found, and so it's a secure context.
const BaseURI = "lootnest://app/"

// Options are the window's.
type Options struct {
	Title               string
	Width, Height       int
	MinWidth, MinHeight int
	Hidden              bool   // created hidden (Lootnest starting in the tray)
	DataDir             string // WebKit's own storage and cache go under it
	Background          uint32 // 0xRRGGBB, shown before the page draws
	Debug               bool   // the inspector and the console on stdout
	// AllowMessage answers whether a call from the page, whose address is
	// given, may reach a bound function.
	AllowMessage func(uri string) bool
	// OnClose is asked when the window's close button is pressed: true
	// keeps the window open (Lootnest decides what closing means).
	OnClose func() bool
}

// WebView is the window and its page.
type WebView struct {
	w        *C.ln_window
	id       cgo.Handle
	opt      Options
	mu       sync.Mutex
	bindings map[string]any

	visible, active, full, max atomic.Bool
}

// New makes the window, hidden until Show or when Options.Hidden is false,
// shown at once. It returns an error when GTK can't start (no display).
func New(opt Options) (*WebView, error) {
	v := &WebView{opt: opt, bindings: map[string]any{}}
	v.id = cgo.NewHandle(v)
	data, cache := filepath.Join(opt.DataDir, "data"), filepath.Join(opt.DataDir, "cache")
	cs := func(s string) *C.char { return C.CString(s) }
	title, cdata, ccache, base := cs(opt.Title), cs(data), cs(cache), cs(BaseURI)
	defer C.free(unsafe.Pointer(title))
	defer C.free(unsafe.Pointer(cdata))
	defer C.free(unsafe.Pointer(ccache))
	defer C.free(unsafe.Pointer(base))
	debug := C.int(0)
	if opt.Debug {
		debug = 1
	}
	rgb := func(shift uint) C.double { return C.double(float64((opt.Background>>shift)&0xff) / 255) }
	v.w = C.ln_new(C.uintptr_t(v.id), title, C.int(opt.Width), C.int(opt.Height), C.int(opt.MinWidth), C.int(opt.MinHeight),
		cdata, ccache, base, rgb(16), rgb(8), rgb(0), debug)
	if v.w == nil {
		v.id.Delete()
		return nil, errors.New("GTK couldn't start: no display to show the window on")
	}
	v.Init(rpcJS)
	if !opt.Hidden {
		C.ln_show(v.w)
	}
	return v, nil
}

// rpcJS is the page's side of Bind: each bound name is a function that
// posts {id, method, params} and returns a promise settled by Go.
const rpcJS = `(function () {
  var RPC = window._rpc = (window._rpc || { nextSeq: 1 });
  window.__lnCall = function (name, args) {
    var seq = RPC.nextSeq++;
    var p = new Promise(function (resolve, reject) { RPC[seq] = { resolve: resolve, reject: reject }; });
    window.webkit.messageHandlers.lootnest.postMessage(JSON.stringify({ id: seq, method: name, params: Array.prototype.slice.call(args) }));
    return p;
  };
})();`

// Init adds a script run at the start of every load of the page, in the
// top frame only.
func (v *WebView) Init(js string) {
	c := C.CString(js)
	defer C.free(unsafe.Pointer(c))
	C.ln_init_script(v.w, c)
}

// Bind makes f callable from the page as window[name](...), returning a
// promise. f takes JSON-decodable arguments and returns nothing, a value,
// an error, or a value and an error.
func (v *WebView) Bind(name string, f any) error {
	fv := reflect.ValueOf(f)
	if fv.Kind() != reflect.Func {
		return errors.New("only functions can be bound")
	}
	if fv.Type().NumOut() > 2 {
		return errors.New("function may only return a value or a value+error")
	}
	v.mu.Lock()
	v.bindings[name] = f
	v.mu.Unlock()
	n, _ := json.Marshal(name)
	v.Init("window[" + string(n) + "] = function () { return window.__lnCall(" + string(n) + ", arguments); };")
	return nil
}

// Dispatch runs f on the GTK thread.
func (v *WebView) Dispatch(f func()) { C.ln_dispatch(C.uintptr_t(cgo.NewHandle(f))) }

// Eval runs a script in the page.
func (v *WebView) Eval(js string) {
	c := C.CString(js)
	defer C.free(unsafe.Pointer(c))
	C.ln_eval(v.w, c)
}

// SetHtml loads the page under BaseURI.
func (v *WebView) SetHtml(html string) {
	c, b := C.CString(html), C.CString(BaseURI)
	defer C.free(unsafe.Pointer(c))
	defer C.free(unsafe.Pointer(b))
	C.ln_load_html(v.w, c, b)
}

// RefreshView shows the window with its page (Windows' WebView2 needs it
// after a hidden start; here it's Show).
func (v *WebView) RefreshView() { C.ln_show(v.w) }

// Show shows the window; Present also brings it to the front.
func (v *WebView) Show()    { C.ln_show(v.w) }
func (v *WebView) Present() { C.ln_present(v.w) }
func (v *WebView) Hide()    { C.ln_hide(v.w) }

// SetTitle and SetSize change the window.
func (v *WebView) SetTitle(t string) {
	c := C.CString(t)
	defer C.free(unsafe.Pointer(c))
	C.ln_set_title(v.w, c)
}
func (v *WebView) SetSize(w, h int) { C.ln_resize(v.w, C.int(w), C.int(h)) }

// SetFullscreen fills the window's monitor, or stops.
func (v *WebView) SetFullscreen(on bool) {
	n := C.int(0)
	if on {
		n = 1
	}
	C.ln_fullscreen(v.w, n)
}

// FullscreenPrimary fills the primary monitor, as TV mode and Big Picture
// on a TV want.
func (v *WebView) FullscreenPrimary() { C.ln_fullscreen_primary(v.w) }

// Visible, Active (it has the keyboard), Fullscreen and Maximized are the
// window's state as GTK last told it; any goroutine may read them.
func (v *WebView) Visible() bool    { return v.visible.Load() }
func (v *WebView) Active() bool     { return v.active.Load() }
func (v *WebView) Fullscreen() bool { return v.full.Load() }
func (v *WebView) Maximized() bool  { return v.max.Load() }

// Window is the GtkWindow.
func (v *WebView) Window() unsafe.Pointer { return unsafe.Pointer(v.w.window) }

// Run runs GTK until Terminate.
func (v *WebView) Run() { C.ln_run() }

// Terminate ends Run; any goroutine may call it.
func (v *WebView) Terminate() { v.Dispatch(func() { C.ln_quit() }) }

// Destroy closes the window.
func (v *WebView) Destroy() {
	C.ln_destroy(v.w)
	v.id.Delete()
}

//export lnIdle
func lnIdle(fn C.uintptr_t) C.gboolean {
	h := cgo.Handle(fn)
	f := h.Value().(func())
	h.Delete()
	f()
	return C.FALSE
}

//export lnState
func lnState(id C.uintptr_t, visible, active, full, max C.int) {
	v := cgo.Handle(id).Value().(*WebView)
	v.visible.Store(visible != 0)
	v.active.Store(active != 0)
	v.full.Store(full != 0)
	v.max.Store(max != 0)
}

//export lnClose
func lnClose(id C.uintptr_t) C.gboolean {
	v := cgo.Handle(id).Value().(*WebView)
	if v.opt.OnClose != nil && v.opt.OnClose() {
		return C.TRUE
	}
	C.ln_quit()
	return C.FALSE
}

//export lnMessage
func lnMessage(id C.uintptr_t, msg, uri *C.char) {
	v := cgo.Handle(id).Value().(*WebView)
	v.message(C.GoString(msg), C.GoString(uri))
}

type rpcMessage struct {
	ID     int               `json:"id"`
	Method string            `json:"method"`
	Params []json.RawMessage `json:"params"`
}

func jsString(s string) string { b, _ := json.Marshal(s); return string(b) }

// message runs a call from the page and settles its promise.
func (v *WebView) message(msg, uri string) {
	if uri != BaseURI || (v.opt.AllowMessage != nil && !v.opt.AllowMessage(uri)) {
		return
	}
	var d rpcMessage
	if err := json.Unmarshal([]byte(msg), &d); err != nil {
		log.Printf("invalid call from the page: %v", err)
		return
	}
	id := strconv.Itoa(d.ID)
	settle := func(ok bool, val string) {
		how := "reject"
		if ok {
			how = "resolve"
		}
		v.Eval("window._rpc[" + id + "] && (window._rpc[" + id + "]." + how + "(" + val + "), window._rpc[" + id + "] = undefined)")
	}
	res, err := v.call(d)
	if err != nil {
		settle(false, jsString(err.Error()))
		return
	}
	b, err := json.Marshal(res)
	if err != nil {
		settle(false, jsString(err.Error()))
		return
	}
	settle(true, string(b))
}

// call runs a bound function with the call's arguments.
func (v *WebView) call(d rpcMessage) (any, error) {
	v.mu.Lock()
	f, ok := v.bindings[d.Method]
	v.mu.Unlock()
	if !ok {
		return nil, nil
	}
	fv := reflect.ValueOf(f)
	ft := fv.Type()
	variadic, numIn := ft.IsVariadic(), ft.NumIn()
	if (variadic && len(d.Params) < numIn-1) || (!variadic && len(d.Params) != numIn) {
		return nil, errors.New("function arguments mismatch")
	}
	args := make([]reflect.Value, 0, len(d.Params))
	for i, raw := range d.Params {
		var t reflect.Type
		if variadic && i >= numIn-1 {
			t = ft.In(numIn - 1).Elem()
		} else {
			t = ft.In(i)
		}
		arg := reflect.New(t)
		if err := json.Unmarshal(raw, arg.Interface()); err != nil {
			return nil, err
		}
		args = append(args, arg.Elem())
	}
	errType := reflect.TypeOf((*error)(nil)).Elem()
	out := fv.Call(args)
	switch len(out) {
	case 0:
		return nil, nil
	case 1:
		if out[0].Type().Implements(errType) {
			if out[0].IsNil() {
				return nil, nil
			}
			return nil, out[0].Interface().(error)
		}
		return out[0].Interface(), nil
	case 2:
		if !out[1].Type().Implements(errType) {
			return nil, fmt.Errorf("second result of %s must be an error", d.Method)
		}
		if !out[1].IsNil() {
			return nil, out[1].Interface().(error)
		}
		return out[0].Interface(), nil
	}
	return nil, errors.New("unexpected results")
}
