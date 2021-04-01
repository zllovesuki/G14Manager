#ifndef WINUTIL_H_
#define WINUTIL_H_

#include <windows.h>
#include <stdexcept>

namespace winutil {

template<typename FuncType>
FuncType getLibraryFunction(std::string const& library, std::string const& funcName) {
	//HINSTANCE dll{LoadLibrary(library.c_str())};
	auto dll = LoadLibrary(library.c_str());
	if (NULL == dll) {
		throw std::invalid_argument(std::string{"failed to load library "} + library);
	}

	// FreeLibrary(dll) ???
	return reinterpret_cast<FuncType>(GetProcAddress(dll, funcName.c_str()));
}

void setWindowTransparancy(HWND, BYTE);

}

#endif
