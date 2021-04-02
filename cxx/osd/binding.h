#pragma once

#include <stddef.h>
#include <stdlib.h>

#ifdef __cplusplus
extern "C"
{
#endif

    extern void *pWindow;
    int NewWindow(int, int);
    void ShowText(char *, int);
    void Hide();

#ifdef __cplusplus
}
#endif