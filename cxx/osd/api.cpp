#include <windows.h>

#include "window.h"
#include "api.hxx"
#include <iostream>

void *fnNewWindow(int height, int width)
{
    Window *pWindow = NULL;
    try
    {
        pWindow = new Window(height, width);
    }
    catch (const std::runtime_error &e)
    {
        std::cerr << e.what() << std::endl;
    }
    return pWindow;
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