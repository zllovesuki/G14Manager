#include <windows.h>
#include <mmdeviceapi.h>
#include <endpointvolume.h>
#include <Functiondiscoverykeys_devpkey.h>

#include "volume.h"

int SetMicrophoneMute(int check_only, int set_muted)
{
    int ret = -1;
    HRESULT hr;
    IMMDeviceEnumerator *pDeviceEnumerator = NULL;
    IMMDevice *pDefaultDevice = NULL;
    IAudioEndpointVolume *pEndpointVolume = NULL;

    hr = CoInitializeEx(NULL, COINIT_MULTITHREADED);
    if (S_OK != hr)
    {
        goto GTFO;
    }

    hr = CoCreateInstance(__uuidof(MMDeviceEnumerator), NULL, CLSCTX_ALL, __uuidof(IMMDeviceEnumerator), (LPVOID *)&pDeviceEnumerator);
    if (S_OK != hr)
    {
        goto GTFO;
    }

    hr = pDeviceEnumerator->GetDefaultAudioEndpoint(eCapture, eConsole, &pDefaultDevice);
    if (S_OK != hr)
    {
        pDeviceEnumerator->Release();
        goto GTFO;
    }
    pDeviceEnumerator->Release();
    pDeviceEnumerator = NULL;

    hr = pDefaultDevice->Activate(__uuidof(IAudioEndpointVolume), CLSCTX_ALL, NULL, (LPVOID *)&pEndpointVolume);
    if (S_OK != hr)
    {
        pDefaultDevice->Release();
        goto GTFO;
    }
    pDefaultDevice->Release();
    pDefaultDevice = NULL;

    if (check_only)
    {
        hr = pEndpointVolume->GetMute(&ret);
    }
    else
    {
        hr = pEndpointVolume->SetMute(set_muted, NULL);
        ret = 0;
    }
    if (S_OK != hr && S_FALSE != hr)
    {
        pEndpointVolume->Release();
        goto GTFO;
    }
    pEndpointVolume->Release();

GTFO:
    CoUninitialize();
    return ret;
}