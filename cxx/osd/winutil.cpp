#include "winutil.h"
#include <windows.h>

#ifndef LWA_ALPHA
#define LWA_ALPHA	0x00000002
#endif

namespace {

char const* user32dll{"user32.dll"};

}

namespace winutil {

void setWindowTransparancy(HWND windowHandle, BYTE alpha) {
	// The following code is now done within CreateWindowEx
	//SetWindowLong(windowHandle, GWL_EXSTYLE, GetWindowLong(windowHandle, GWL_EXSTYLE) | WS_EX_LAYERED);

	//typedef void (*FuncType)(HWND, COLORREF, BYTE, DWORD);	//works in debug build, but not release
	//typedef bool (CALLBACK* FuncType)(HWND, COLORREF, BYTE, DWORD);	// CALLBACK attribute is required
	using FuncType=bool (CALLBACK*)(HWND, COLORREF, BYTE, DWORD);
	auto setLayWinAttributes = getLibraryFunction<FuncType>(user32dll, "SetLayeredWindowAttributes");
	setLayWinAttributes(windowHandle, RGB(255, 255, 255), alpha, LWA_ALPHA);
}

}
