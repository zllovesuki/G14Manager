#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN // Exclude rarely-used stuff from Windows headers
#endif

#include <Windows.h>
#include <SetupAPI.h>
#include <cfgmgr32.h>
#include <iostream>

#include <device.h>

#define MAX_LEN 50
const char *nvidiaMfgName = "NVIDIA";
const GUID GUID_DEVINTERFACE_DISPLAY_ADAPTER = {
    0x5B45201D, 0xF2F2, 0x4F3B, {0x85, 0xBB, 0x30, 0xFF, 0x1F, 0x95, 0x35, 0x99}};

int changeState(HDEVINFO deviceInfoSet, PSP_DEVINFO_DATA pDeviceInfoData, DWORD state, DWORD scope)
{
    SP_PROPCHANGE_PARAMS params;
    ZeroMemory(&params, sizeof(SP_PROPCHANGE_PARAMS));

    params.ClassInstallHeader.cbSize = sizeof(SP_CLASSINSTALL_HEADER);
    params.ClassInstallHeader.InstallFunction = DIF_PROPERTYCHANGE;
    params.StateChange = state;
    params.Scope = scope;
    params.HwProfile = 0;

    if (!SetupDiSetClassInstallParamsA(deviceInfoSet, pDeviceInfoData, &params.ClassInstallHeader, sizeof(SP_PROPCHANGE_PARAMS)))
    {
        std::cerr << "gpu: SetupDiSetClassInstallParamsA failed: " << GetLastError() << std::endl;
        return 0;
    }
    if (!SetupDiCallClassInstaller(DIF_PROPERTYCHANGE, deviceInfoSet, pDeviceInfoData))
    {
        std::cerr << "gpu: SetupDiCallClassInstaller failed: " << GetLastError() << std::endl;
        return 0;
    }

    return 1;
}

int restartGPU(void)
{
    int ret = 0;
    HDEVINFO deviceInfoSet;
    deviceInfoSet = SetupDiGetClassDevsA(&GUID_DEVINTERFACE_DISPLAY_ADAPTER, NULL, NULL, DIGCF_DEVICEINTERFACE | DIGCF_PRESENT);
    if (INVALID_HANDLE_VALUE == deviceInfoSet)
    {
        std::cerr << "gpu: SetupDiGetClassDevsA error: " << GetLastError() << std::endl;
        return 0;
    }

    SP_DEVINFO_DATA deviceInfoData;
    ZeroMemory(&deviceInfoData, sizeof(SP_DEVINFO_DATA));
    deviceInfoData.cbSize = sizeof(SP_DEVINFO_DATA);

    unsigned long status = 0;
    unsigned long problem = 0;
    char mfgName[MAX_LEN] = {0};
    int foundDevice = 0;

    int deviceMemberIndex = 0;
    while (SetupDiEnumDeviceInfo(deviceInfoSet, deviceMemberIndex, &deviceInfoData))
    {
        deviceMemberIndex++;
        deviceInfoData.cbSize = sizeof(deviceInfoData);

        if (CR_SUCCESS != CM_Get_DevNode_Status(&status, &problem, deviceInfoData.DevInst, 0))
        {
            std::cerr << "gpu: CM_Get_DevNode_Status error: " << GetLastError() << std::endl;
            goto GTFO;
        }

        SetupDiGetDeviceRegistryPropertyA(deviceInfoSet, &deviceInfoData, SPDRP_MFG, 0, (PBYTE)mfgName, MAX_LEN, NULL);

        if (strncmp(mfgName, nvidiaMfgName, MAX_LEN) == 0)
        {
            foundDevice = 1;
            break;
        }
    }

    if (foundDevice == 0)
    {
        std::cerr << "gpu: Cannot found NVIDIA graphics card" << std::endl;
        goto GTFO;
    }

    if (!(status & DN_STARTED))
    {
        std::cerr << "gpu: Dedicated graphics card is not enabled" << std::endl;
        goto GTFO;
    }

    if (!changeState(deviceInfoSet, &deviceInfoData, DICS_DISABLE, DICS_FLAG_CONFIGSPECIFIC))
    {
        std::cerr << "gpu: Error disabling device globally" << std::endl;
        goto GTFO;
    }

    Sleep(1000);

    if (!changeState(deviceInfoSet, &deviceInfoData, DICS_ENABLE, DICS_FLAG_GLOBAL))
    {
        std::cerr << "gpu: Error enabling device globally" << std::endl;
        goto GTFO;
    }
    if (!changeState(deviceInfoSet, &deviceInfoData, DICS_ENABLE, DICS_FLAG_CONFIGSPECIFIC))
    {
        std::cerr << "gpu: Error enabling device with profile" << std::endl;
        goto GTFO;
    }

    ret = 1;

GTFO:
    if (!SetupDiDestroyDeviceInfoList(deviceInfoSet))
    {
        std::cerr << "gpu: SetupDiDestroyDeviceInfoList error: " << GetLastError() << std::endl;
    }

    return ret;
}
