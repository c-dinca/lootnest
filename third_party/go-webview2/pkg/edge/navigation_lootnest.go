//go:build windows
// +build windows

package edge

// Lootnest patch: NavigationStarting and NewWindowRequested, so a window
// can keep to one site (a page elsewhere opens in the user's browser) and
// keep a link that wants a new window in its own.

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// ICoreWebView2NavigationStartingEventArgs

type iCoreWebView2NavigationStartingEventArgsVtbl struct {
	_IUnknownVtbl
	GetUri             ComProc
	GetIsUserInitiated ComProc
	GetIsRedirected    ComProc
	GetRequestHeaders  ComProc
	GetCancel          ComProc
	PutCancel          ComProc
	GetNavigationId    ComProc
}

type iCoreWebView2NavigationStartingEventArgs struct {
	vtbl *iCoreWebView2NavigationStartingEventArgsVtbl
}

type iCoreWebView2NavigationStartingEventHandlerImpl interface {
	_IUnknownImpl
	NavigationStarting(sender *ICoreWebView2, args *iCoreWebView2NavigationStartingEventArgs) uintptr
}

type iCoreWebView2NavigationStartingEventHandlerVtbl struct {
	_IUnknownVtbl
	Invoke ComProc
}

type iCoreWebView2NavigationStartingEventHandler struct {
	vtbl *iCoreWebView2NavigationStartingEventHandlerVtbl
	impl iCoreWebView2NavigationStartingEventHandlerImpl
}

func _NavigationStartingQueryInterface(this *iCoreWebView2NavigationStartingEventHandler, refiid, object uintptr) uintptr {
	return this.impl.QueryInterface(refiid, object)
}

func _NavigationStartingAddRef(this *iCoreWebView2NavigationStartingEventHandler) uintptr {
	return this.impl.AddRef()
}

func _NavigationStartingRelease(this *iCoreWebView2NavigationStartingEventHandler) uintptr {
	return this.impl.Release()
}

func _NavigationStartingInvoke(this *iCoreWebView2NavigationStartingEventHandler, sender *ICoreWebView2, args *iCoreWebView2NavigationStartingEventArgs) uintptr {
	return this.impl.NavigationStarting(sender, args)
}

var iCoreWebView2NavigationStartingEventHandlerFn = iCoreWebView2NavigationStartingEventHandlerVtbl{
	_IUnknownVtbl{
		NewComProc(_NavigationStartingQueryInterface),
		NewComProc(_NavigationStartingAddRef),
		NewComProc(_NavigationStartingRelease),
	},
	NewComProc(_NavigationStartingInvoke),
}

func newICoreWebView2NavigationStartingEventHandler(impl iCoreWebView2NavigationStartingEventHandlerImpl) *iCoreWebView2NavigationStartingEventHandler {
	return &iCoreWebView2NavigationStartingEventHandler{vtbl: &iCoreWebView2NavigationStartingEventHandlerFn, impl: impl}
}

// ICoreWebView2NewWindowRequestedEventArgs

type iCoreWebView2NewWindowRequestedEventArgsVtbl struct {
	_IUnknownVtbl
	GetUri             ComProc
	PutNewWindow       ComProc
	GetNewWindow       ComProc
	PutHandled         ComProc
	GetHandled         ComProc
	GetIsUserInitiated ComProc
	GetDeferral        ComProc
	GetWindowFeatures  ComProc
}

type iCoreWebView2NewWindowRequestedEventArgs struct {
	vtbl *iCoreWebView2NewWindowRequestedEventArgsVtbl
}

type iCoreWebView2NewWindowRequestedEventHandlerImpl interface {
	_IUnknownImpl
	NewWindowRequested(sender *ICoreWebView2, args *iCoreWebView2NewWindowRequestedEventArgs) uintptr
}

type iCoreWebView2NewWindowRequestedEventHandlerVtbl struct {
	_IUnknownVtbl
	Invoke ComProc
}

type iCoreWebView2NewWindowRequestedEventHandler struct {
	vtbl *iCoreWebView2NewWindowRequestedEventHandlerVtbl
	impl iCoreWebView2NewWindowRequestedEventHandlerImpl
}

func _NewWindowRequestedQueryInterface(this *iCoreWebView2NewWindowRequestedEventHandler, refiid, object uintptr) uintptr {
	return this.impl.QueryInterface(refiid, object)
}

func _NewWindowRequestedAddRef(this *iCoreWebView2NewWindowRequestedEventHandler) uintptr {
	return this.impl.AddRef()
}

func _NewWindowRequestedRelease(this *iCoreWebView2NewWindowRequestedEventHandler) uintptr {
	return this.impl.Release()
}

func _NewWindowRequestedInvoke(this *iCoreWebView2NewWindowRequestedEventHandler, sender *ICoreWebView2, args *iCoreWebView2NewWindowRequestedEventArgs) uintptr {
	return this.impl.NewWindowRequested(sender, args)
}

var iCoreWebView2NewWindowRequestedEventHandlerFn = iCoreWebView2NewWindowRequestedEventHandlerVtbl{
	_IUnknownVtbl{
		NewComProc(_NewWindowRequestedQueryInterface),
		NewComProc(_NewWindowRequestedAddRef),
		NewComProc(_NewWindowRequestedRelease),
	},
	NewComProc(_NewWindowRequestedInvoke),
}

func newICoreWebView2NewWindowRequestedEventHandler(impl iCoreWebView2NewWindowRequestedEventHandlerImpl) *iCoreWebView2NewWindowRequestedEventHandler {
	return &iCoreWebView2NewWindowRequestedEventHandler{vtbl: &iCoreWebView2NewWindowRequestedEventHandlerFn, impl: impl}
}

// argURI reads a URI an event's GetUri gives, and frees it.
func argURI(get ComProc, args unsafe.Pointer) string {
	var uri *uint16
	_, _, _ = get.Call(uintptr(args), uintptr(unsafe.Pointer(&uri)))
	if uri == nil {
		return ""
	}
	s := windows.UTF16PtrToString(uri)
	windows.CoTaskMemFree(unsafe.Pointer(uri))
	return s
}

// NavigationStarting asks NavigationFilter about every page the window
// would show; one it refuses isn't loaded.
func (e *Chromium) NavigationStarting(_ *ICoreWebView2, args *iCoreWebView2NavigationStartingEventArgs) uintptr {
	if e.NavigationFilter == nil {
		return 0
	}
	if !e.NavigationFilter(argURI(args.vtbl.GetUri, unsafe.Pointer(args))) {
		_, _, _ = args.vtbl.PutCancel.Call(uintptr(unsafe.Pointer(args)), 1)
	}
	return 0
}

// NewWindowRequested keeps a link that wants a new window in this one:
// NewWindowCallback is given its address, and no window opens.
func (e *Chromium) NewWindowRequested(_ *ICoreWebView2, args *iCoreWebView2NewWindowRequestedEventArgs) uintptr {
	if e.NewWindowCallback == nil {
		return 0
	}
	uri := argURI(args.vtbl.GetUri, unsafe.Pointer(args))
	_, _, _ = args.vtbl.PutHandled.Call(uintptr(unsafe.Pointer(args)), 1)
	e.NewWindowCallback(uri)
	return 0
}

// addNavigationHandlers registers the two when either is wanted.
func (e *Chromium) addNavigationHandlers() {
	var token _EventRegistrationToken
	if e.NavigationFilter != nil {
		_, _, _ = e.webview.vtbl.AddNavigationStarting.Call(
			uintptr(unsafe.Pointer(e.webview)),
			uintptr(unsafe.Pointer(e.navigationStarting)),
			uintptr(unsafe.Pointer(&token)),
		)
	}
	if e.NewWindowCallback != nil {
		_, _, _ = e.webview.vtbl.AddNewWindowRequested.Call(
			uintptr(unsafe.Pointer(e.webview)),
			uintptr(unsafe.Pointer(e.newWindowRequested)),
			uintptr(unsafe.Pointer(&token)),
		)
	}
}
