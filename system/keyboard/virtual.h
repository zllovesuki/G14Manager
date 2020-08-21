#include <windows.h>

// https://forum.openframeworks.cc/t/how-to-emulate-a-physical-key-press/28438/11
// https://docs.microsoft.com/en-us/windows/win32/api/winuser/ns-winuser-keybdinput

int send_key_press(unsigned short key_code) {
    INPUT myInput;

    myInput.type = INPUT_KEYBOARD;
    myInput.ki.time = 0;
    myInput.ki.wVk = 0;
    myInput.ki.dwExtraInfo = 0;

    myInput.ki.dwFlags = KEYEVENTF_SCANCODE;
    myInput.ki.wScan = key_code;

    int ret;

    ret = SendInput(1, &myInput, sizeof(INPUT));
    if (ret == 0) {
        return 1;
    }

    myInput.ki.dwFlags = KEYEVENTF_SCANCODE | KEYEVENTF_KEYUP;
    ret = SendInput(1, &myInput, sizeof(INPUT));
    if (ret == 0) {
        return 1;
    }

    return 0;
}