#include "api.hxx"
#include "binding.h"

#ifdef __cplusplus
extern "C"
{
#endif

    void *pWindow = NULL;

    int NewWindow(int height, int width)
    {
        pWindow = fnNewWindow(height, width);
        return pWindow != NULL;
    }

    void ShowText(char *text, int fontSize)
    {
        fnShowText(pWindow, text, fontSize);
    }

    void Hide()
    {
        fnHide(pWindow);
    }
#ifdef __cplusplus
}
#endif