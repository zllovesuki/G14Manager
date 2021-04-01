#ifndef WINDOW_H_
#define WINDOW_H_

#include <windows.h>
#include <map>
#include <string>

struct Window
{
    Window(int, int);
    void show();
    void hide();
    void setText(std::string const &, int);

    LRESULT CALLBACK processWindowEvent(HWND, UINT, WPARAM, LPARAM);

private:
    int getScreenCenterX() const;
    int getScreenCenterY() const;
    void makeTransparent();
    void makeRounded();
    void repaint();
    void paint(HWND);

    HINSTANCE instanceHandle;

    HWND windowHandle{};
    int fontSize;
    int windowWidth;
    int windowHeight;
    std::string text;
};

#endif
