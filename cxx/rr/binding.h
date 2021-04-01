#pragma once

#include <stddef.h>

#ifdef __cplusplus
extern "C"
{
#endif

    extern void *pDisplay;

    int GetDisplay();
    int CycleRefreshRate();
    int GetCurrentRefreshRate();
    void ReleaseDisplay();

#ifdef __cplusplus
}
#endif