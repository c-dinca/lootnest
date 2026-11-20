//go:build windows
// +build windows

package edge

// Lootnest patch: DownloadStarting (ICoreWebView2_4), so a window can put
// a download where it wants it and hide the browser's download list.
// The interfaces' layouts follow WebView2.h of the WebView2 SDK
// 1.0.2903.40.

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

type iCoreWebView2_4Vtbl struct {
	iCoreWebView2_3Vtbl
	AddFrameCreated        ComProc
	RemoveFrameCreated     ComProc
	AddDownloadStarting    ComProc
	RemoveDownloadStarting ComProc
}

// ICoreWebView2_4 is the WebView with downloads it can be told about.
type ICoreWebView2_4 struct {
	vtbl *iCoreWebView2_4Vtbl
}

// GetICoreWebView2_4 is nil on a WebView2 runtime too old to have it.
func (i *ICoreWebView2) GetICoreWebView2_4() *ICoreWebView2_4 {
	var result *ICoreWebView2_4
	iid := NewGUID("{20d02d59-6df2-42dc-bd06-f98a694b1302}")
	_, _, _ = i.vtbl.QueryInterface.Call(
		uintptr(unsafe.Pointer(i)),
		uintptr(unsafe.Pointer(iid)),
		uintptr(unsafe.Pointer(&result)))
	return result
}

type iCoreWebView2DownloadStartingEventArgsVtbl struct {
	_IUnknownVtbl
	GetDownloadOperation ComProc
	GetCancel            ComProc
	PutCancel            ComProc
	GetResultFilePath    ComProc
	PutResultFilePath    ComProc
	GetHandled           ComProc
	PutHandled           ComProc
	GetDeferral          ComProc
}

type iCoreWebView2DownloadStartingEventArgs struct {
	vtbl *iCoreWebView2DownloadStartingEventArgsVtbl
}

type iCoreWebView2DownloadOperationVtbl struct {
	_IUnknownVtbl
	AddBytesReceivedChanged       ComProc
	RemoveBytesReceivedChanged    ComProc
	AddEstimatedEndTimeChanged    ComProc
	RemoveEstimatedEndTimeChanged ComProc
	AddStateChanged               ComProc
	RemoveStateChanged            ComProc
	GetUri                        ComProc
	GetContentDisposition         ComProc
	GetMimeType                   ComProc
	GetTotalBytesToReceive        ComProc
	GetBytesReceived              ComProc
	GetEstimatedEndTime           ComProc
	GetResultFilePath             ComProc
	GetState                      ComProc
	GetInterruptReason            ComProc
	Cancel                        ComProc
	Pause                         ComProc
	Resume                        ComProc
	GetCanResume                  ComProc
}

type iCoreWebView2DownloadOperation struct {
	vtbl *iCoreWebView2DownloadOperationVtbl
}

type iCoreWebView2DownloadStartingEventHandlerImpl interface {
	_IUnknownImpl
	DownloadStarting(sender *ICoreWebView2, args *iCoreWebView2DownloadStartingEventArgs) uintptr
}

type iCoreWebView2DownloadStartingEventHandlerVtbl struct {
	_IUnknownVtbl
	Invoke ComProc
}

type iCoreWebView2DownloadStartingEventHandler struct {
	vtbl *iCoreWebView2DownloadStartingEventHandlerVtbl
	impl iCoreWebView2DownloadStartingEventHandlerImpl
}

func _DownloadStartingQueryInterface(this *iCoreWebView2DownloadStartingEventHandler, refiid, object uintptr) uintptr {
	return this.impl.QueryInterface(refiid, object)
}

func _DownloadStartingAddRef(this *iCoreWebView2DownloadStartingEventHandler) uintptr {
	return this.impl.AddRef()
}

func _DownloadStartingRelease(this *iCoreWebView2DownloadStartingEventHandler) uintptr {
	return this.impl.Release()
}

func _DownloadStartingInvoke(this *iCoreWebView2DownloadStartingEventHandler, sender *ICoreWebView2, args *iCoreWebView2DownloadStartingEventArgs) uintptr {
	return this.impl.DownloadStarting(sender, args)
}

var iCoreWebView2DownloadStartingEventHandlerFn = iCoreWebView2DownloadStartingEventHandlerVtbl{
	_IUnknownVtbl{
		NewComProc(_DownloadStartingQueryInterface),
		NewComProc(_DownloadStartingAddRef),
		NewComProc(_DownloadStartingRelease),
	},
	NewComProc(_DownloadStartingInvoke),
}

func newICoreWebView2DownloadStartingEventHandler(impl iCoreWebView2DownloadStartingEventHandlerImpl) *iCoreWebView2DownloadStartingEventHandler {
	return &iCoreWebView2DownloadStartingEventHandler{vtbl: &iCoreWebView2DownloadStartingEventHandlerFn, impl: impl}
}

// Download is a download about to start, for DownloadFilter.
type Download struct {
	URI   string // where it comes from
	Path  string // where WebView2 would put it (the Downloads folder and the file's name)
	Total int64  // its size, or 0 when not known
}

// DownloadStarting asks DownloadFilter where a download goes. With an
// empty path it goes where WebView2 would put it; hide keeps the
// browser's download list from showing; cancel stops it.
func (e *Chromium) DownloadStarting(_ *ICoreWebView2, args *iCoreWebView2DownloadStartingEventArgs) uintptr {
	if e.DownloadFilter == nil {
		return 0
	}
	a := unsafe.Pointer(args)
	d := Download{Path: argURI(args.vtbl.GetResultFilePath, a)}
	var op *iCoreWebView2DownloadOperation
	_, _, _ = args.vtbl.GetDownloadOperation.Call(uintptr(a), uintptr(unsafe.Pointer(&op)))
	if op != nil {
		d.URI = argURI(op.vtbl.GetUri, unsafe.Pointer(op))
		var total int64
		_, _, _ = op.vtbl.GetTotalBytesToReceive.Call(uintptr(unsafe.Pointer(op)), uintptr(unsafe.Pointer(&total)))
		if total > 0 {
			d.Total = total
		}
		_, _, _ = op.vtbl.Release.Call(uintptr(unsafe.Pointer(op)))
	}
	path, hide, cancel := e.DownloadFilter(d)
	if cancel {
		_, _, _ = args.vtbl.PutCancel.Call(uintptr(a), 1)
		return 0
	}
	if path != "" {
		if p, err := windows.UTF16PtrFromString(path); err == nil {
			_, _, _ = args.vtbl.PutResultFilePath.Call(uintptr(a), uintptr(unsafe.Pointer(p)))
		}
	}
	if hide {
		_, _, _ = args.vtbl.PutHandled.Call(uintptr(a), 1)
	}
	return 0
}

// addDownloadHandler registers DownloadStarting when it's wanted and the
// runtime has it; an older runtime keeps its own downloads as they are.
func (e *Chromium) addDownloadHandler() {
	if e.DownloadFilter == nil {
		return
	}
	wv4 := e.webview.GetICoreWebView2_4()
	if wv4 == nil {
		return
	}
	var token _EventRegistrationToken
	_, _, _ = wv4.vtbl.AddDownloadStarting.Call(
		uintptr(unsafe.Pointer(wv4)),
		uintptr(unsafe.Pointer(e.downloadStarting)),
		uintptr(unsafe.Pointer(&token)),
	)
}
