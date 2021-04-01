#ifndef DISPLAY_H_
#define DISPLAY_H_

#include <windows.h>
#include <list>
#include <set>
#include <memory>
#include <iosfwd>

struct Display
{
    static std::list<Display> getDisplays();

    bool isAMD() const;
    bool isPrimary() const;
    bool isActive() const;
    DEVMODEA getDisplaySettings() const;

    DWORD getRefreshRate() const;
    bool setRefreshRate(DWORD);
    std::set<DWORD> getSupportedRefreshRates() const;

    std::ostream &write(std::ostream &out) const;

private:
    DISPLAY_DEVICEA displayDevice;

    Display(DISPLAY_DEVICEA);
};

std::ostream &operator<<(std::ostream &lhv, Display const &display);

#endif
