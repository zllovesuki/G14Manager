#include "api.hxx"
#include "binding.h"

#ifdef __cplusplus
extern "C"
{
#endif

    void *pDisplay = NULL;

    int GetDisplay(void)
    {
        pDisplay = fnGetDisplay();
        return pDisplay != NULL;
    }

    int CycleRefreshRate()
    {
        return fnCycleRefreshRate(pDisplay);
    }

    int GetCurrentRefreshRate()
    {
        return fnGetCurrentRefreshRate(pDisplay);
    }

    void ReleaseDisplay()
    {
        fnReleaseDisplay(pDisplay);
    }
#ifdef __cplusplus
}
#endif