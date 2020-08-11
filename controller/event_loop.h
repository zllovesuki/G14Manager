#include <windows.h>

static int event_loop(uintptr_t handle_ptr)
{
    HWND *hwnd = (HWND *)handle_ptr;
    MSG m;
    int r;

    while (*hwnd)
    {
        r = GetMessage(&m, NULL, 0, 0);
        if (!r)
            return m.wParam;
        else if (r < 0)
            return -1;
        if (!IsDialogMessage(*hwnd, &m))
        {
            TranslateMessage(&m);
            DispatchMessage(&m);
        }
    }
    return 0;
}