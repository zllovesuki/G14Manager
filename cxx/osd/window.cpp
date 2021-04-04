#include "Window.h"

#include <windows.h>
#include <functional>
#include <map>
#include <stdexcept>
#include <string>
#include <memory>
#include <iostream>

namespace
{

    char const *className{"G14Manager"};
    char const *windowTitle{"OSD for various info"};

    std::map<HWND, std::shared_ptr<Window>> handleWindowMap{};

    // retrieves all windows api messages and delegates them to the proper window instance
    // we cannot directly use the member function as we need a c-style function pointer
    LRESULT CALLBACK processWinApiMessage(HWND windowHandle, UINT msg, WPARAM wParam, LPARAM lParam)
    {
        auto it = handleWindowMap.find(windowHandle);
        if (it != handleWindowMap.end())
        {
            return it->second->processWindowEvent(windowHandle, msg, wParam, lParam);
        }
        return DefWindowProc(windowHandle, msg, wParam, lParam);
    }

    void registerWindowClass(HINSTANCE instanceHandle)
    {
        WNDCLASSEX wc;
        wc.cbSize = sizeof(WNDCLASSEX);
        wc.style = 0;
        wc.lpfnWndProc = processWinApiMessage;
        wc.cbClsExtra = 0;
        wc.cbWndExtra = 0;
        wc.hInstance = instanceHandle;
        wc.hIcon = LoadIcon(NULL, IDI_APPLICATION);
        wc.hCursor = LoadCursor(NULL, IDC_ARROW);
        //wc.hbrBackground = (HBRUSH)(COLOR_WINDOW);	// default background color
        wc.hbrBackground = CreateSolidBrush(RGB(0, 0, 0)); // black background
        wc.lpszMenuName = NULL;
        wc.lpszClassName = className;
        wc.hIconSm = LoadIcon(NULL, IDI_APPLICATION);

        if (!RegisterClassEx(&wc))
        {
            throw std::runtime_error("osd: Window registration failed");
        }
    }

}

void Window::paint(HWND hwnd)
{
    PAINTSTRUCT ps{};
    HDC hdc{BeginPaint(hwnd, &ps)};

    HFONT font{CreateFont(fontSize, 0, 0, 0, FW_REGULAR, 0, 0, 0, 0, 0, 0, CLEARTYPE_QUALITY, 0, DISPLAY_FONT)};
    SelectObject(hdc, font);

    SetTextAlign(hdc, TA_CENTER | TA_BASELINE);
    SetBkMode(hdc, TRANSPARENT);
    SetTextColor(hdc, RGB(255, 255, 255));
    TextOut(hdc, windowWidth / 2, windowHeight / 2 + fontSize / 4, text.c_str(), text.size());

    DeleteObject(font);
    EndPaint(hwnd, &ps);
}

LRESULT CALLBACK Window::processWindowEvent(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam)
{
    switch (msg)
    {
    case WM_CLOSE:
        DestroyWindow(hwnd);
        break;
    case WM_DESTROY:
        handleWindowMap.erase(hwnd);
        PostQuitMessage(0);
        break;
    case WM_PAINT:
        paint(hwnd);
        break;
    default:
        return DefWindowProc(hwnd, msg, wParam, lParam);
    }
    return 0;
}

void Window::setText(std::string const &text, int fontSize)
{
    this->text = text;
    this->fontSize = fontSize;
    repaint();
}

void Window::repaint()
{
    InvalidateRect(windowHandle, NULL, TRUE);
}

Window::Window(int width, int height) : windowWidth{width},
                                        windowHeight{height}, text{}
{
    instanceHandle = GetModuleHandle("");

    registerWindowClass(instanceHandle);

    // WS_EX_CLIENTEDGE - normal window
    // WS_EX_TOOLWINDOW - window without taskbar and alt+tab entry
    // WS_EX_TOPMOST - always on top
    // WS_EX_NOACTIVATE - do not steal focus
    // WS_EX_LAYERED - layered style, this enables transparency support
    // ---
    // WS_OVEERLAPPEDWINDOW - with border
    // WS_POPUP - borderless
    // WS_CHILD -
    windowHandle = CreateWindowEx(WS_EX_TOPMOST | WS_EX_NOACTIVATE | WS_EX_TOOLWINDOW | WS_EX_LAYERED, className, windowTitle, WS_POPUP,
                                  getScreenCenterX(), getScreenCenterY(), width, height, NULL, NULL, instanceHandle, NULL);
    if (!windowHandle)
    {
        throw std::runtime_error("osd: Failed to create window");
    }
    handleWindowMap[windowHandle] = std::shared_ptr<Window>{this};

    // // rounded corners
    // SetWindowRgn(windowHandle, CreateRoundRectRgn(0, 0, windowWidth, windowHeight, 50, 50), false);
    // // transparent background
    SetLayeredWindowAttributes(windowHandle, RGB(255, 255, 255), 255, LWA_ALPHA);

    // preload font so it will load faster when we actually show text
    HFONT preload{CreateFont(5, 0, 0, 0, FW_REGULAR, 0, 0, 0, 0, 0, 0, CLEARTYPE_QUALITY, 0, DISPLAY_FONT)};
    DeleteObject(preload);
}

void Window::show()
{
    ShowWindow(windowHandle, SW_NORMAL);
    UpdateWindow(windowHandle);
}

void Window::hide()
{
    ShowWindow(windowHandle, SW_HIDE);
    UpdateWindow(windowHandle);
}

int Window::getScreenCenterX() const
{
    return (GetSystemMetrics(SM_CXSCREEN) - windowWidth) / 2;
}

int Window::getScreenCenterY() const
{
    return (GetSystemMetrics(SM_CYSCREEN) - windowHeight) * 1 / 10;
}
