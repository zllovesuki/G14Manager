#pragma once

#include <stddef.h>

#ifdef __cplusplus
extern "C"
{
#endif

    void *NewWindow(int, int);
    void ShowText(void *, char *, int);
    void Hide(void *);

#ifdef __cplusplus
}
#endif