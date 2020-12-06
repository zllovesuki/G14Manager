#pragma once

#include "MatrixController.hxx"

typedef struct _API_WRAPPER {
	WINUSB_INTERFACE_HANDLE WinusbHandle;
	HANDLE                  DeviceHandle;
	MatrixController*		mc;
	char					devicePath[MAX_PATH + 1];
} API_WRAPPER, *PAPI_WRAPPER;