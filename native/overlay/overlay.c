// Lootnest's in-game FPS counter, loaded only into games without an
// anti-cheat (Lootnest decides; app/overlay.go). It hooks the game's
// Present, counts frames there, and, when Lootnest says so through a small
// shared memory block, draws "144 FPS 6.9 ms" in a corner of the frame:
// with ClearView rectangles in Direct3D 11, Clear rectangles in Direct3D 9
// and scissored clears in OpenGL, so it needs no shaders, fonts or state
// of its own. Direct3D 12 frames are only counted (its games have no
// exclusive fullscreen, so Lootnest's own window shows over them). It
// reads and writes nothing of the game but the frame it adds the number to.
//
// Built by tools/overlay/build.sh with MinGW-w64 into
// internal/app/overlay/lootnest-overlay64.dll; MinHook (BSD 2-clause,
// third_party/minhook) does the hooking.

#define COBJMACROS
#define WIN32_LEAN_AND_MEAN
#define INITGUID
#include <windows.h>
#include <stdint.h>
#include <string.h>
#include <d3d11.h>
#include <d3d11_1.h>
#include <dxgi1_2.h>
#include <d3d12.h>
#include <d3d9.h>
#include <GL/gl.h>
#include "MinHook.h"

// ---------------------------------------------------------------- shared --
// Laid out as core.OverlayShare: 64 bytes, "Local\LootnestOverlay-<pid>".
typedef struct {
  uint32_t magic;            // 'LNOV'
  uint32_t version;          // 1
  volatile int32_t on;       // Lootnest: draw the number
  volatile int32_t corner;   // Lootnest: 0 top left, 1 top right, 2 bottom left, 3 bottom right
  volatile int32_t fps;      // here: frames in the last second
  volatile int32_t frame_us; // here: a frame's time, in microseconds
  volatile int32_t api;      // here: 1 Direct3D 11, 2 Direct3D 12, 3 Direct3D 9, 4 OpenGL
  volatile int32_t drawn;    // here: frames the number was drawn on
  volatile int64_t beat;     // here: QueryPerformanceCounter at the last frame
  volatile int32_t error;    // here: what failed, for Lootnest's log (LN_ERR_*)
  volatile int32_t hooked;   // here: bit per hook in place (1 DXGI, 2 D3D9, 4 OpenGL)
  int32_t reserved[6];
} ln_share;

#define LN_MAGIC 0x564F4E4Cu
#define LN_VERSION 1
enum { API_NONE, API_D3D11, API_D3D12, API_D3D9, API_GL };
enum { LN_ERR_NONE, LN_ERR_SHARE, LN_ERR_MINHOOK, LN_ERR_DXGI, LN_ERR_D3D9, LN_ERR_GL, LN_ERR_RTV };

static ln_share *share;
static ln_share fallback; // until the block is open, counts go here
static LARGE_INTEGER qpf;

static ln_share *sh(void) { return share ? share : &fallback; }

// ----------------------------------------------------------------- frames --
#define RING 1024
static int64_t ring[RING];
static int ring_n, ring_at;
static int show_fps, show_us;
static int64_t show_at;
static CRITICAL_SECTION count_cs;

// note_frame counts one frame and refreshes the number shown four times a
// second, so it doesn't flicker.
static void note_frame(int api) {
  LARGE_INTEGER now;
  QueryPerformanceCounter(&now);
  EnterCriticalSection(&count_cs);
  ring[ring_at] = now.QuadPart;
  ring_at = (ring_at + 1) % RING;
  if (ring_n < RING) ring_n++;
  int64_t from = now.QuadPart - qpf.QuadPart;
  int n = 0;
  int64_t oldest = now.QuadPart;
  for (int i = 0; i < ring_n; i++) {
    int64_t t = ring[(ring_at - 1 - i + RING) % RING];
    if (t <= from) break;
    n++;
    oldest = t;
  }
  if (now.QuadPart - show_at > qpf.QuadPart / 4) {
    show_at = now.QuadPart;
    show_fps = n;
    show_us = n > 1 ? (int)((now.QuadPart - oldest) * 1000000 / qpf.QuadPart / (n - 1)) : 0;
  }
  ln_share *s = sh();
  s->fps = show_fps;
  s->frame_us = show_us;
  s->beat = now.QuadPart;
  if (s->api == API_NONE || (s->api == API_D3D12 && api != API_D3D12)) s->api = api;
  LeaveCriticalSection(&count_cs);
}

// ------------------------------------------------------------------- text --
// A 5x7 pixel font for what the counter says.
static const char *glyph(char c) {
  switch (c) {
  case '0': return "01110100011001110101110011000101110";
  case '1': return "00100011000010000100001000010001110";
  case '2': return "01110100010000100010001000100011111";
  case '3': return "11111000100010000010000011000101110";
  case '4': return "00010001100101010010111110001000010";
  case '5': return "11111100001111000001000011000101110";
  case '6': return "00110010001000011110100011000101110";
  case '7': return "11111000010001000100010000100001000";
  case '8': return "01110100011000101110100011000101110";
  case '9': return "01110100011000101111000010001001100";
  case 'F': return "11111100001000011110100001000010000";
  case 'P': return "11110100011000111110100001000010000";
  case 'S': return "01111100001000001110000010000111110";
  case 'm': return "00000000001101010101101011000110001";
  case 's': return "00000000000111110000011100000111110";
  case '.': return "00000000000000000000000000110001100";
  default: return NULL; // a space
  }
}

typedef struct { LONG l, t, r, b; } box;
#define MAX_BOXES 512

// layout makes the boxes to fill: the panel first, then the letters, for a
// frame w by h. It returns how many.
static int layout(int w, int h, box *out, int *panel) {
  char text[48];
  int fps = show_fps, us = show_us;
  if (us > 0)
    wsprintfA(text, "%d FPS  %d.%d ms", fps, us / 1000, (us % 1000) / 100);
  else
    wsprintfA(text, "%d FPS", fps);
  int px = h / 360;
  if (px < 2) px = 2;
  if (px > 8) px = 8;
  int len = lstrlenA(text);
  int tw = len * 6 * px - px, th = 7 * px;
  int pad = 3 * px, margin = 4 * px;
  int bw = tw + 2 * pad, bh = th + 2 * pad;
  int corner = sh()->corner;
  int x = (corner == 1 || corner == 3) ? w - margin - bw : margin;
  int y = (corner == 2 || corner == 3) ? h - margin - bh : margin;
  if (x < 0 || y < 0 || bw > w || bh > h) return 0;
  int n = 0;
  out[n++] = (box){x, y, x + bw, y + bh};
  *panel = 1;
  for (int i = 0; i < len && n < MAX_BOXES; i++) {
    const char *g = glyph(text[i]);
    if (!g) continue;
    int gx = x + pad + i * 6 * px, gy = y + pad;
    for (int row = 0; row < 7; row++) {
      for (int col = 0; col < 5; col++) {
        if (g[row * 5 + col] != '1') continue;
        int run = col;
        while (run + 1 < 5 && g[row * 5 + run + 1] == '1') run++;
        if (n < MAX_BOXES) out[n++] = (box){gx + col * px, gy + row * px, gx + (run + 1) * px, gy + (row + 1) * px};
        col = run;
      }
    }
  }
  return n;
}

// Obsidian ground, Ice Grey letters (the toast's colours).
static const float ground[4] = {0x11 / 255.0f, 0x12 / 255.0f, 0x17 / 255.0f, 1};
static const float ink[4] = {0xDF / 255.0f, 0xE5 / 255.0f, 0xEB / 255.0f, 1};

static int drawing(void) { return share && share->on && show_fps > 0; }

// ------------------------------------------------------------------ DXGI --
typedef HRESULT(STDMETHODCALLTYPE *present_fn)(IDXGISwapChain *, UINT, UINT);
typedef HRESULT(STDMETHODCALLTYPE *present1_fn)(IDXGISwapChain1 *, UINT, UINT, const DXGI_PRESENT_PARAMETERS *);
static present_fn real_present;
static present1_fn real_present1;

static DXGI_FORMAT typed(DXGI_FORMAT f) {
  switch (f) {
  case DXGI_FORMAT_R8G8B8A8_TYPELESS: return DXGI_FORMAT_R8G8B8A8_UNORM;
  case DXGI_FORMAT_B8G8R8A8_TYPELESS: return DXGI_FORMAT_B8G8R8A8_UNORM;
  case DXGI_FORMAT_R10G10B10A2_TYPELESS: return DXGI_FORMAT_R10G10B10A2_UNORM;
  case DXGI_FORMAT_R16G16B16A16_TYPELESS: return DXGI_FORMAT_R16G16B16A16_FLOAT;
  default: return f;
  }
}

static void draw_d3d11(IDXGISwapChain *sc, ID3D11Device *dev) {
  ID3D11DeviceContext *ctx = NULL;
  ID3D11DeviceContext1 *ctx1 = NULL;
  ID3D11Texture2D *bb = NULL;
  ID3D11RenderTargetView *rtv = NULL;
  ID3D11Device_GetImmediateContext(dev, &ctx);
  if (!ctx) return;
  if (FAILED(ID3D11DeviceContext_QueryInterface(ctx, &IID_ID3D11DeviceContext1, (void **)&ctx1))) goto out;
  if (FAILED(IDXGISwapChain_GetBuffer(sc, 0, &IID_ID3D11Texture2D, (void **)&bb))) goto out;
  D3D11_TEXTURE2D_DESC td;
  ID3D11Texture2D_GetDesc(bb, &td);
  D3D11_RENDER_TARGET_VIEW_DESC rd = {0};
  rd.Format = typed(td.Format);
  rd.ViewDimension = td.SampleDesc.Count > 1 ? D3D11_RTV_DIMENSION_TEXTURE2DMS : D3D11_RTV_DIMENSION_TEXTURE2D;
  if (FAILED(ID3D11Device_CreateRenderTargetView(dev, (ID3D11Resource *)bb, &rd, &rtv))) {
    sh()->error = LN_ERR_RTV;
    goto out;
  }
  box b[MAX_BOXES];
  int panel = 0, n = layout((int)td.Width, (int)td.Height, b, &panel);
  if (n > 0) {
    ID3D11DeviceContext1_ClearView(ctx1, (ID3D11View *)rtv, ground, (const D3D11_RECT *)b, 1);
    ID3D11DeviceContext1_ClearView(ctx1, (ID3D11View *)rtv, ink, (const D3D11_RECT *)(b + 1), n - 1);
    sh()->drawn++;
  }
out:
  if (rtv) ID3D11RenderTargetView_Release(rtv);
  if (bb) ID3D11Texture2D_Release(bb);
  if (ctx1) ID3D11DeviceContext1_Release(ctx1);
  ID3D11DeviceContext_Release(ctx);
}

static void on_dxgi(IDXGISwapChain *sc) {
  ID3D11Device *d11 = NULL;
  ID3D12Device *d12 = NULL;
  if (SUCCEEDED(IDXGISwapChain_GetDevice(sc, &IID_ID3D11Device, (void **)&d11)) && d11) {
    note_frame(API_D3D11);
    if (drawing()) draw_d3d11(sc, d11);
    ID3D11Device_Release(d11);
    return;
  }
  if (SUCCEEDED(IDXGISwapChain_GetDevice(sc, &IID_ID3D12Device, (void **)&d12)) && d12) {
    ID3D12Device_Release(d12);
  }
  note_frame(API_D3D12); // counted; Lootnest's window shows the number
}

// One Present may call another inside (Present1, PresentEx): the frame
// counts once, on the outermost, per thread.
static __thread int depth;

static HRESULT STDMETHODCALLTYPE hook_present(IDXGISwapChain *sc, UINT sync, UINT flags) {
  int test = (flags & DXGI_PRESENT_TEST) != 0;
  if (!test && depth++ == 0) on_dxgi(sc);
  HRESULT hr = real_present(sc, sync, flags);
  if (!test) depth--;
  return hr;
}

static HRESULT STDMETHODCALLTYPE hook_present1(IDXGISwapChain1 *sc, UINT sync, UINT flags, const DXGI_PRESENT_PARAMETERS *p) {
  int test = (flags & DXGI_PRESENT_TEST) != 0;
  if (!test && depth++ == 0) on_dxgi((IDXGISwapChain *)sc);
  HRESULT hr = real_present1(sc, sync, flags, p);
  if (!test) depth--;
  return hr;
}

static LRESULT CALLBACK dummy_proc(HWND h, UINT m, WPARAM w, LPARAM l) { return DefWindowProcW(h, m, w, l); }

static HWND dummy_window(void) {
  WNDCLASSEXW wc;
  ZeroMemory(&wc, sizeof wc);
  wc.cbSize = sizeof wc;
  wc.lpfnWndProc = dummy_proc;
  wc.hInstance = GetModuleHandleW(NULL);
  wc.lpszClassName = L"LootnestOverlayProbe";
  RegisterClassExW(&wc);
  return CreateWindowExW(0, wc.lpszClassName, L"", WS_OVERLAPPEDWINDOW, 0, 0, 8, 8, NULL, NULL, wc.hInstance, NULL);
}

typedef HRESULT(WINAPI *create11_fn)(IDXGIAdapter *, D3D_DRIVER_TYPE, HMODULE, UINT, const D3D_FEATURE_LEVEL *, UINT, UINT,
                                     const DXGI_SWAP_CHAIN_DESC *, IDXGISwapChain **, ID3D11Device **, D3D_FEATURE_LEVEL *,
                                     ID3D11DeviceContext **);

// hook_dxgi finds IDXGISwapChain's Present (and Present1) through a
// throwaway swap chain of its own, and hooks them. Every swap chain in the
// process shares that code.
static int hook_dxgi(void) {
  HMODULE d3d11 = LoadLibraryW(L"d3d11.dll");
  if (!d3d11) return 0;
  create11_fn create = (create11_fn)GetProcAddress(d3d11, "D3D11CreateDeviceAndSwapChain");
  if (!create) return 0;
  HWND w = dummy_window();
  if (!w) return 0;
  DXGI_SWAP_CHAIN_DESC d = {0};
  d.BufferDesc.Width = 8;
  d.BufferDesc.Height = 8;
  d.BufferDesc.Format = DXGI_FORMAT_R8G8B8A8_UNORM;
  d.SampleDesc.Count = 1;
  d.BufferUsage = DXGI_USAGE_RENDER_TARGET_OUTPUT;
  d.BufferCount = 1;
  d.OutputWindow = w;
  d.Windowed = TRUE;
  d.SwapEffect = DXGI_SWAP_EFFECT_DISCARD;
  IDXGISwapChain *sc = NULL;
  ID3D11Device *dev = NULL;
  ID3D11DeviceContext *ctx = NULL;
  HRESULT hr = create(NULL, D3D_DRIVER_TYPE_HARDWARE, NULL, 0, NULL, 0, D3D11_SDK_VERSION, &d, &sc, &dev, NULL, &ctx);
  if (FAILED(hr)) hr = create(NULL, D3D_DRIVER_TYPE_WARP, NULL, 0, NULL, 0, D3D11_SDK_VERSION, &d, &sc, &dev, NULL, &ctx);
  int ok = 0;
  if (SUCCEEDED(hr) && sc) {
    void **vt = *(void ***)sc;
    if (MH_CreateHook(vt[8], (void *)hook_present, (void **)&real_present) == MH_OK && MH_EnableHook(vt[8]) == MH_OK) ok = 1;
    IDXGISwapChain1 *sc1 = NULL;
    if (SUCCEEDED(IDXGISwapChain_QueryInterface(sc, &IID_IDXGISwapChain1, (void **)&sc1)) && sc1) {
      void **vt1 = *(void ***)sc1;
      if (MH_CreateHook(vt1[22], (void *)hook_present1, (void **)&real_present1) == MH_OK) MH_EnableHook(vt1[22]);
      IDXGISwapChain1_Release(sc1);
    }
  }
  if (sc) IDXGISwapChain_Release(sc);
  if (ctx) ID3D11DeviceContext_Release(ctx);
  if (dev) ID3D11Device_Release(dev);
  DestroyWindow(w);
  return ok;
}

// ------------------------------------------------------------------ D3D9 --
typedef HRESULT(STDMETHODCALLTYPE *present9_fn)(IDirect3DDevice9 *, const RECT *, const RECT *, HWND, const RGNDATA *);
typedef HRESULT(STDMETHODCALLTYPE *present9ex_fn)(IDirect3DDevice9Ex *, const RECT *, const RECT *, HWND, const RGNDATA *, DWORD);
static present9_fn real_present9;
static present9ex_fn real_present9ex;

static void draw_d3d9(IDirect3DDevice9 *dev) {
  IDirect3DSurface9 *bb = NULL, *old = NULL;
  if (FAILED(IDirect3DDevice9_GetBackBuffer(dev, 0, 0, D3DBACKBUFFER_TYPE_MONO, &bb)) || !bb) return;
  D3DSURFACE_DESC sd;
  IDirect3DSurface9_GetDesc(bb, &sd);
  D3DVIEWPORT9 vp;
  DWORD scissor = 0;
  IDirect3DDevice9_GetViewport(dev, &vp);
  IDirect3DDevice9_GetRenderState(dev, D3DRS_SCISSORTESTENABLE, &scissor);
  IDirect3DDevice9_GetRenderTarget(dev, 0, &old);
  IDirect3DDevice9_SetRenderTarget(dev, 0, bb);
  D3DVIEWPORT9 full = {0, 0, sd.Width, sd.Height, 0, 1};
  IDirect3DDevice9_SetViewport(dev, &full);
  IDirect3DDevice9_SetRenderState(dev, D3DRS_SCISSORTESTENABLE, FALSE);
  box b[MAX_BOXES];
  int panel = 0, n = layout((int)sd.Width, (int)sd.Height, b, &panel);
  if (n > 0) {
    D3DCOLOR g = D3DCOLOR_ARGB(255, 0x11, 0x12, 0x17), k = D3DCOLOR_ARGB(255, 0xDF, 0xE5, 0xEB);
    IDirect3DDevice9_Clear(dev, 1, (const D3DRECT *)b, D3DCLEAR_TARGET, g, 1.0f, 0);
    IDirect3DDevice9_Clear(dev, n - 1, (const D3DRECT *)(b + 1), D3DCLEAR_TARGET, k, 1.0f, 0);
    sh()->drawn++;
  }
  IDirect3DDevice9_SetRenderTarget(dev, 0, old);
  IDirect3DDevice9_SetViewport(dev, &vp);
  IDirect3DDevice9_SetRenderState(dev, D3DRS_SCISSORTESTENABLE, scissor);
  if (old) IDirect3DSurface9_Release(old);
  IDirect3DSurface9_Release(bb);
}

static HRESULT STDMETHODCALLTYPE hook_present9(IDirect3DDevice9 *dev, const RECT *a, const RECT *b, HWND w, const RGNDATA *r) {
  if (depth++ == 0) {
    note_frame(API_D3D9);
    if (drawing()) draw_d3d9(dev);
  }
  HRESULT hr = real_present9(dev, a, b, w, r);
  depth--;
  return hr;
}

static HRESULT STDMETHODCALLTYPE hook_present9ex(IDirect3DDevice9Ex *dev, const RECT *a, const RECT *b, HWND w, const RGNDATA *r, DWORD f) {
  if (depth++ == 0) {
    note_frame(API_D3D9);
    if (drawing()) draw_d3d9((IDirect3DDevice9 *)dev);
  }
  HRESULT hr = real_present9ex(dev, a, b, w, r, f);
  depth--;
  return hr;
}

typedef IDirect3D9 *(WINAPI *create9_fn)(UINT);
typedef HRESULT(WINAPI *create9ex_fn)(UINT, IDirect3D9Ex **);

static int hook_d3d9(void) {
  HMODULE m = GetModuleHandleW(L"d3d9.dll");
  if (!m) return 0;
  create9ex_fn create = (create9ex_fn)GetProcAddress(m, "Direct3DCreate9Ex");
  if (!create) return 0;
  IDirect3D9Ex *d3d = NULL;
  if (FAILED(create(D3D_SDK_VERSION, &d3d)) || !d3d) return 0;
  HWND w = dummy_window();
  D3DPRESENT_PARAMETERS pp = {0};
  pp.Windowed = TRUE;
  pp.SwapEffect = D3DSWAPEFFECT_DISCARD;
  pp.hDeviceWindow = w;
  pp.BackBufferFormat = D3DFMT_UNKNOWN;
  IDirect3DDevice9Ex *dev = NULL;
  int ok = 0;
  if (SUCCEEDED(IDirect3D9Ex_CreateDeviceEx(d3d, D3DADAPTER_DEFAULT, D3DDEVTYPE_HAL, w,
                                            D3DCREATE_SOFTWARE_VERTEXPROCESSING | D3DCREATE_DISABLE_DRIVER_MANAGEMENT, &pp, NULL, &dev)) && dev) {
    void **vt = *(void ***)dev;
    // A plain device and an Ex one share Present's code in d3d9.dll.
    if (MH_CreateHook(vt[17], (void *)hook_present9, (void **)&real_present9) == MH_OK && MH_EnableHook(vt[17]) == MH_OK) ok = 1;
    if (MH_CreateHook(vt[121], (void *)hook_present9ex, (void **)&real_present9ex) == MH_OK) MH_EnableHook(vt[121]);
    IDirect3DDevice9Ex_Release(dev);
  }
  IDirect3D9Ex_Release(d3d);
  if (w) DestroyWindow(w);
  return ok;
}

// ---------------------------------------------------------------- OpenGL --
typedef BOOL(WINAPI *swap_fn)(HDC);
static swap_fn real_swap;
typedef void(APIENTRY *bindfb_fn)(GLenum, GLuint);
static bindfb_fn glBindFramebuffer_;
// OpenGL's functions, from the game's own opengl32.dll (not imported, so
// games without OpenGL never load it for us).
static struct {
  GLboolean(APIENTRY *IsEnabled)(GLenum);
  void(APIENTRY *GetIntegerv)(GLenum, GLint *);
  void(APIENTRY *GetFloatv)(GLenum, GLfloat *);
  void(APIENTRY *GetBooleanv)(GLenum, GLboolean *);
  void(APIENTRY *Enable)(GLenum);
  void(APIENTRY *Disable)(GLenum);
  void(APIENTRY *ColorMask)(GLboolean, GLboolean, GLboolean, GLboolean);
  void(APIENTRY *ClearColor)(GLfloat, GLfloat, GLfloat, GLfloat);
  void(APIENTRY *Scissor)(GLint, GLint, GLsizei, GLsizei);
  void(APIENTRY *Clear)(GLbitfield);
  HGLRC(WINAPI *GetCurrentContext)(void);
  PROC(WINAPI *GetProcAddress)(LPCSTR);
} gl;

#define GL_DRAW_FRAMEBUFFER_BINDING_ 0x8CA6
#define GL_DRAW_FRAMEBUFFER_ 0x8CA9

static void draw_gl(HDC dc) {
  HWND w = WindowFromDC(dc);
  RECT rc;
  if (!w || !GetClientRect(w, &rc)) return;
  int W = rc.right - rc.left, H = rc.bottom - rc.top;
  if (!glBindFramebuffer_) glBindFramebuffer_ = (bindfb_fn)gl.GetProcAddress("glBindFramebuffer");
  GLint fb = 0, sbox[4];
  GLfloat cc[4];
  GLboolean mask[4];
  GLboolean sc = gl.IsEnabled(GL_SCISSOR_TEST);
  gl.GetIntegerv(GL_SCISSOR_BOX, sbox);
  gl.GetFloatv(GL_COLOR_CLEAR_VALUE, cc);
  gl.GetBooleanv(GL_COLOR_WRITEMASK, mask);
  if (glBindFramebuffer_) {
    gl.GetIntegerv(GL_DRAW_FRAMEBUFFER_BINDING_, &fb);
    glBindFramebuffer_(GL_DRAW_FRAMEBUFFER_, 0);
  }
  gl.Enable(GL_SCISSOR_TEST);
  gl.ColorMask(GL_TRUE, GL_TRUE, GL_TRUE, GL_TRUE);
  box b[MAX_BOXES];
  int panel = 0, n = layout(W, H, b, &panel);
  for (int i = 0; i < n; i++) {
    const float *c = i == 0 ? ground : ink;
    gl.ClearColor(c[0], c[1], c[2], c[3]);
    gl.Scissor(b[i].l, H - b[i].b, b[i].r - b[i].l, b[i].b - b[i].t); // GL counts from the bottom
    gl.Clear(GL_COLOR_BUFFER_BIT);
  }
  if (n > 0) sh()->drawn++;
  if (glBindFramebuffer_) glBindFramebuffer_(GL_DRAW_FRAMEBUFFER_, (GLuint)fb);
  gl.ColorMask(mask[0], mask[1], mask[2], mask[3]);
  gl.ClearColor(cc[0], cc[1], cc[2], cc[3]);
  gl.Scissor(sbox[0], sbox[1], sbox[2], sbox[3]);
  if (!sc) gl.Disable(GL_SCISSOR_TEST);
}

static BOOL WINAPI hook_swap(HDC dc) {
  note_frame(API_GL);
  if (drawing() && gl.GetCurrentContext()) draw_gl(dc);
  return real_swap(dc);
}

static int hook_gl(void) {
  HMODULE m = GetModuleHandleW(L"opengl32.dll");
  if (!m) return 0;
  void *swap = (void *)GetProcAddress(m, "wglSwapBuffers");
  if (!swap) return 0;
#define GLF(f, n) { FARPROC p_ = GetProcAddress(m, n); if (!p_) return 0; memcpy(&gl.f, &p_, sizeof p_); }
  GLF(IsEnabled, "glIsEnabled"); GLF(GetIntegerv, "glGetIntegerv"); GLF(GetFloatv, "glGetFloatv");
  GLF(GetBooleanv, "glGetBooleanv"); GLF(Enable, "glEnable"); GLF(Disable, "glDisable");
  GLF(ColorMask, "glColorMask"); GLF(ClearColor, "glClearColor"); GLF(Scissor, "glScissor");
  GLF(Clear, "glClear"); GLF(GetCurrentContext, "wglGetCurrentContext"); GLF(GetProcAddress, "wglGetProcAddress");
#undef GLF
  return MH_CreateHook(swap, (void *)hook_swap, (void **)&real_swap) == MH_OK && MH_EnableHook(swap) == MH_OK;
}

// ------------------------------------------------------------------ start --
static DWORD WINAPI start(LPVOID unused) {
  (void)unused;
  wchar_t name[64];
  wsprintfW(name, L"Local\\LootnestOverlay-%lu", GetCurrentProcessId());
  for (int i = 0; i < 50 && !share; i++) {
    HANDLE m = OpenFileMappingW(FILE_MAP_READ | FILE_MAP_WRITE, FALSE, name);
    if (m) {
      ln_share *s = (ln_share *)MapViewOfFile(m, FILE_MAP_READ | FILE_MAP_WRITE, 0, 0, sizeof(ln_share));
      if (s && s->magic == LN_MAGIC && s->version == LN_VERSION) share = s;
      // The view keeps the block alive; the handle isn't needed.
      CloseHandle(m);
    }
    if (!share) Sleep(100);
  }
  if (!share) return 0; // Lootnest made none: count nothing, draw nothing
  if (MH_Initialize() != MH_OK) {
    share->error = LN_ERR_MINHOOK;
    return 0;
  }
  // The game may load its graphics after Lootnest came in: look for a
  // minute, hooking each one as it appears.
  int dxgi = 0, d9 = 0, gl = 0;
  for (int i = 0; i < 120 && !(dxgi && d9 && gl); i++) {
    if (!dxgi && GetModuleHandleW(L"dxgi.dll")) {
      dxgi = hook_dxgi() ? 1 : -1;
      if (dxgi < 0) share->error = LN_ERR_DXGI;
    }
    if (!d9 && GetModuleHandleW(L"d3d9.dll")) {
      d9 = hook_d3d9() ? 1 : -1;
      if (d9 < 0) share->error = LN_ERR_D3D9;
    }
    if (!gl && GetModuleHandleW(L"opengl32.dll")) {
      gl = hook_gl() ? 1 : -1;
      if (gl < 0) share->error = LN_ERR_GL;
    }
    share->hooked = (dxgi > 0 ? 1 : 0) | (d9 > 0 ? 2 : 0) | (gl > 0 ? 4 : 0);
    if (share->api != API_NONE) break; // frames arrive: this is the one
    Sleep(500);
  }
  return 0;
}

BOOL WINAPI DllMain(HINSTANCE dll, DWORD reason, LPVOID unused) {
  (void)unused;
  if (reason == DLL_PROCESS_ATTACH) {
    DisableThreadLibraryCalls(dll);
    QueryPerformanceFrequency(&qpf);
    InitializeCriticalSection(&count_cs);
    HANDLE t = CreateThread(NULL, 0, start, NULL, 0, NULL);
    if (t) CloseHandle(t);
  }
  return TRUE;
}
