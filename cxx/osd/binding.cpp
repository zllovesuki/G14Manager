#include "api.hxx"
#include "binding.h"

#ifdef __cplusplus
extern "C"
{
#endif
    void *NewWindow(int height, int width)
    {
        return fnNewWindow(height, width);
    }

    void ShowText(void *pWindow, char *text, int fontSize)
    {
        fnShowText(pWindow, text, fontSize);
    }

    void Hide(void *pWindow)
    {
        fnHide(pWindow);
    }
#ifdef __cplusplus
}
#endif