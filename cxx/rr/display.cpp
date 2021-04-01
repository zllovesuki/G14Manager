#include "display.h"

#include <windows.h>
#include <list>
#include <set>
#include <memory>
#include <ostream>

std::list<Display> Display::getDisplays()
{
    std::list<Display> displays;
    //int index{};
    DISPLAY_DEVICEA displayDevice{};
    displayDevice.cb = sizeof(DISPLAY_DEVICEA);

    for (int i{}; EnumDisplayDevicesA(NULL, i, &displayDevice, 0); ++i)
    {
        displays.push_back(Display{displayDevice});
        displayDevice = DISPLAY_DEVICEA{};
        displayDevice.cb = sizeof(DISPLAY_DEVICEA);
    }
    return displays;
}

Display::Display(DISPLAY_DEVICEA dp) : displayDevice(dp)
{
}

bool Display::isPrimary() const
{
    return displayDevice.StateFlags & DISPLAY_DEVICE_PRIMARY_DEVICE;
}

bool Display::isAMD() const
{
    std::string device(displayDevice.DeviceString);
    return device.find("AMD") != std::string::npos;
}

bool Display::isActive() const
{
    return displayDevice.StateFlags & DISPLAY_DEVICE_ACTIVE;
}

DWORD Display::getRefreshRate() const
{
    return getDisplaySettings().dmDisplayFrequency;
}

bool Display::setRefreshRate(DWORD refreshRate)
{
    auto deviceMode = getDisplaySettings();
    deviceMode.dmDisplayFrequency = refreshRate;
    return ChangeDisplaySettingsA(&deviceMode, 0) == DISP_CHANGE_SUCCESSFUL;
}

std::set<DWORD> Display::getSupportedRefreshRates() const
{
    std::set<DWORD> refreshRates{};
    DEVMODEA deviceMode{};
    deviceMode.dmSize = sizeof(DEVMODEA);
    //int index{};

    DEVMODEA activeDeviceMode = getDisplaySettings();
    //while(EnumDisplaySettings(displayDevice.DeviceName, index++, &deviceMode)) {
    for (int i{}; EnumDisplaySettingsA(displayDevice.DeviceName, i, &deviceMode); ++i)
    {
        // We skip different resolutions as there is a slight possibility that one
        // resolution supports other refresh rates than the current.
        //std::cout << deviceMode.dmPelsWidth << 'x' << deviceMode.dmPelsHeight << '=';
        //std::cout << activeDeviceMode.dmPelsWidth << 'x' << activeDeviceMode.dmPelsHeight << '\n';
        if (activeDeviceMode.dmPelsWidth != deviceMode.dmPelsWidth || activeDeviceMode.dmPelsHeight != deviceMode.dmPelsHeight)
        {
            continue;
        }
        refreshRates.insert(deviceMode.dmDisplayFrequency);
    }
    return refreshRates;
}

DEVMODEA Display::getDisplaySettings() const
{
    DEVMODEA deviceMode{};
    deviceMode.dmSize = sizeof(DEVMODEA);
    EnumDisplaySettingsA(displayDevice.DeviceName, ENUM_CURRENT_SETTINGS, &deviceMode);
    return deviceMode;
}

std::ostream &Display::write(std::ostream &out) const
{
    return out << displayDevice.DeviceName << " - "
               << displayDevice.DeviceString << " - " << getRefreshRate() << " Hz"
               << " - " << (isActive() ? "active" : "inactive")
               << (isPrimary() ? " - primary" : "");
}

std::ostream &operator<<(std::ostream &lhv, Display const &rhv)
{
    return rhv.write(lhv);
}
