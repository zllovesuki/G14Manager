#include <windows.h>

#include "virtual.h"

// https://forum.openframeworks.cc/t/how-to-emulate-a-physical-key-press/28438/11
// https://docs.microsoft.com/en-us/windows/win32/api/winuser/ns-winuser-keybdinput

int SendKeyPress(unsigned short key_code)
{
    INPUT input;
    KEYBDINPUT kbInput;
    ZeroMemory(&input, sizeof(INPUT));
    ZeroMemory(&kbInput, sizeof(KEYBDINPUT));

    input.type = INPUT_KEYBOARD;

    kbInput.wScan = key_code;
    kbInput.dwFlags = KEYEVENTF_SCANCODE;
    input.ki = kbInput;

    int ret;

    ret = SendInput(1, &input, sizeof(INPUT));
    if (ret == 0)
    {
        return 1;
    }

    kbInput.dwFlags = KEYEVENTF_SCANCODE | KEYEVENTF_KEYUP;
    input.ki = kbInput;

    ret = SendInput(1, &input, sizeof(INPUT));
    if (ret == 0)
    {
        return 1;
    }

    return 0;
}