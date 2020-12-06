#include <windows.h>
#include <mmdeviceapi.h>
#include <endpointvolume.h>
#include <Functiondiscoverykeys_devpkey.h>

#include "volume.h"

int SetMicrophoneMute(int check_only, int set_muted)
{
    HRESULT hr;
    int ret = 0;

    hr = CoInitialize(NULL);
    if (S_OK != hr)
    {
        return -1;
    }

    IMMDeviceEnumerator *deviceEnumerator = NULL;
    hr = CoCreateInstance(__uuidof(MMDeviceEnumerator), NULL, CLSCTX_ALL, __uuidof(IMMDeviceEnumerator), (LPVOID *)&deviceEnumerator);
    if (S_OK != hr)
    {
        CoUninitialize();
        return -1;
    }

    IMMDevice *defaultDevice = NULL;
    hr = deviceEnumerator->GetDefaultAudioEndpoint(eCapture, eConsole, &defaultDevice);
    if (S_OK != hr)
    {
        deviceEnumerator->Release();
        CoUninitialize();
        return -1;
    }
    deviceEnumerator->Release();
    deviceEnumerator = NULL;

    IAudioEndpointVolume *endpointVolume = NULL;
    hr = defaultDevice->Activate(__uuidof(IAudioEndpointVolume), CLSCTX_ALL, NULL, (LPVOID *)&endpointVolume);
    if (S_OK != hr)
    {
        defaultDevice->Release();
        CoUninitialize();
        return -1;
    }
    defaultDevice->Release();
    defaultDevice = NULL;

    if (check_only)
    {
        hr = endpointVolume->GetMute(&ret);
    }
    else
    {
        hr = endpointVolume->SetMute(set_muted, NULL);
        ret = 0;
    }
    if (S_OK != hr && S_FALSE != hr)
    {
        endpointVolume->Release();
        CoUninitialize();
        return -1;
    }
    endpointVolume->Release();

    CoUninitialize();
    return ret;
}