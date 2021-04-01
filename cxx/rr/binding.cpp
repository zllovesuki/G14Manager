#include "api.hxx"
#include "binding.h"

#ifdef __cplusplus
extern "C"
{
#endif

    void *GetDisplay(void)
    {
        return fnGetDisplay();
    }

    int CycleRefreshRate(void *p)
    {
        return fnCycleRefreshRate(p);
    }

    int GetCurrentRefreshRate(void *p)
    {
        return fnGetCurrentRefreshRate(p);
    }
#ifdef __cplusplus
}
#endif