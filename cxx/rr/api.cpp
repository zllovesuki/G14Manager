#include "display.h"
#include "api.hxx"

#include <algorithm>
#include <iostream>
#include <iterator>

DWORD nextRefreshRate(Display &display, std::set<DWORD> rates)
{
    auto rate = std::find(rates.begin(), rates.end(), display.getRefreshRate());
    if (rate == rates.end())
    {
        rate = rates.begin();
    }
    else
    {
        rate++;
        if (rate == rates.end())
        {
            rate = rates.begin();
        }
    }
    return *rate;
}

void *fnGetDisplay(void)
{
    auto displays = Display::getDisplays();
    std::cout << "Available devices: \n";
    std::copy(displays.begin(), displays.end(), std::ostream_iterator<Display>{std::cout, "\n"});

    auto it = std::find_if(displays.begin(), displays.end(), [](Display const &d) { return d.isPrimary() && d.isAMD(); });
    if (it != displays.end())
    {
        auto refreshRates = (*it).getSupportedRefreshRates();
        std::cout << "Using:\n"
                  << *it << "\nSupported Refresh Rates: ";
        std::copy(refreshRates.begin(), refreshRates.end(), std::ostream_iterator<DWORD>{std::cout, " Hz "});
        std::cout << std::endl;

        Display *pDisplay = (Display *)calloc(1, sizeof(Display));
        *pDisplay = (*it);

        return pDisplay;
    }
    std::cout << "No primary display found";
    return NULL;
}

int fnGetCurrentRefreshRate(void *p)
{
    Display *pDisplay = static_cast<Display *>(p);
    return pDisplay->getRefreshRate() & INT_MAX;
}

int fnCycleRefreshRate(void *p)
{
    Display *pDisplay = static_cast<Display *>(p);
    DWORD n = nextRefreshRate(*pDisplay, pDisplay->getSupportedRefreshRates());
    if (pDisplay->setRefreshRate(n))
    {
        return n & INT_MAX;
    }
    else
    {
        return 0;
    }
}