#include <windows.h>

#include "window.h"
#include "api.hxx"
#include <iostream>

void *fnNewWindow(int height, int width)
{
    Window *window = new Window(height, width);
    return window;
}

void fnShowText(void *pWindow, char *text, int fontSize)
{
    Window *window = static_cast<Window *>(pWindow);
    window->setText(text, fontSize);
    window->show();
}

void fnHide(void *pWindow)
{
    Window *window = static_cast<Window *>(pWindow);
    window->hide();
}

void fnGG(void)
{
    MSG msg{};
    while (GetMessage(&msg, NULL, 0, 0) > 0)
    {
        TranslateMessage(&msg);
        DispatchMessage(&msg);
    }
}