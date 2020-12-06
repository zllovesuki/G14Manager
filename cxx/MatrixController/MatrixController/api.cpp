// api.cpp : Defines the exported functions for the DLL.
//

#include "pch.hxx"
#include "api.hxx"

bool findDevicePath(char* devicePath) {
	HDEVINFO deviceInfoSet;
	deviceInfoSet = SetupDiGetClassDevsA(&GUID_DEVINTERFACE_USB_DEVICE, NULL, NULL, DIGCF_DEVICEINTERFACE);
	if (INVALID_HANDLE_VALUE == deviceInfoSet) {
		std::cerr << "MatrixController: SetupDiGetClassDevsA error: " << GetLastError() << std::endl;
		return false;
	}

	bool foundInterface = false;
	SP_DEVINFO_DATA deviceInfoData;
	ZeroMemory(&deviceInfoData, sizeof(SP_DEVINFO_DATA));
	deviceInfoData.cbSize = sizeof(SP_DEVINFO_DATA);

	int deviceMemberIndex = 0;
	while (SetupDiEnumDeviceInfo(deviceInfoSet, deviceMemberIndex, &deviceInfoData)) {
		deviceMemberIndex++;
		deviceInfoData.cbSize = sizeof(deviceInfoData);

		char c_deviceID[MAX_PATH + 1];
		CM_Get_Device_IDA(deviceInfoData.DevInst, c_deviceID, MAX_PATH, 0);

		std::string deviceID(c_deviceID);
		size_t found = deviceID.find(MatrixController::MATRIX_DEVICE_VEND_PROD);

		if (found == std::string::npos) {
			continue;
		}

		SP_DEVICE_INTERFACE_DATA deviceInterfaceData;
		ZeroMemory(&deviceInterfaceData, sizeof(SP_DEVICE_INTERFACE_DATA));
		deviceInterfaceData.cbSize = sizeof(SP_DEVICE_INTERFACE_DATA);

		int interfaceMemberIndex = 0;
		// This step is questionable. Are we getting the actual interface? Or are we just getting the parent device?
		while (SetupDiEnumDeviceInterfaces(deviceInfoSet, &deviceInfoData, &GUID_DEVINTERFACE_USB_DEVICE, interfaceMemberIndex, &deviceInterfaceData)) {
			interfaceMemberIndex++;
			deviceInterfaceData.cbSize = sizeof(deviceInterfaceData);

			ULONG requiredSize;
			SetupDiGetDeviceInterfaceDetailA(deviceInfoSet, &deviceInterfaceData, NULL, 0, &requiredSize, NULL);

			// Windows system programming at its finest
			PSP_DEVICE_INTERFACE_DETAIL_DATA_A interfaceDetailData = reinterpret_cast<PSP_DEVICE_INTERFACE_DETAIL_DATA_A>(calloc(requiredSize, sizeof(char)));
			if (interfaceDetailData == NULL) {
				std::cerr << "MatrixController: calloc() failed for PSP_DEVICE_INTERFACE_DETAIL_DATA_A" << std::endl;
				goto GTFO;
			}
			interfaceDetailData->cbSize = sizeof(SP_DEVICE_INTERFACE_DETAIL_DATA_A);

			if (SetupDiGetDeviceInterfaceDetailA(deviceInfoSet, &deviceInterfaceData, interfaceDetailData, requiredSize, NULL, NULL)) {
				std::cout << "MatrixController: Device " << interfaceDetailData->DevicePath << " found" << std::endl;

				size_t pathLen = std::strlen(interfaceDetailData->DevicePath);
				std::memcpy(devicePath, interfaceDetailData->DevicePath, pathLen);

				foundInterface = true;
			}

			free(interfaceDetailData);
		}

		if (ERROR_NO_MORE_ITEMS != GetLastError()) {
			std::cerr << "MatrixController: SetupDiEnumDeviceInterfaces error: " << GetLastError() << std::endl;
			return false;
		}
	}

	if (ERROR_NO_MORE_ITEMS != GetLastError()) {
		std::cerr << "MatrixController: SetupDiEnumDeviceInfo error: " << GetLastError() << std::endl;
		return false;
	}

GTFO:
	if (!SetupDiDestroyDeviceInfoList(deviceInfoSet)) {
		std::cerr << "MatrixController: SetupDiDestroyDeviceInfoList error: " << GetLastError() << std::endl;
		return false;
	}

	return foundInterface;
}

PAPI_WRAPPER fnNewController(void) {
	PAPI_WRAPPER wrapper = reinterpret_cast<PAPI_WRAPPER>(calloc(sizeof(API_WRAPPER), 1));
	if (wrapper == NULL) {
		std::cerr << "MatrixController: calloc() failed for API_WRAPPER" << std::endl;
		return NULL;
	}

	std::cout << "MatrixController: Attempting to find AniMe Matrix Device" << std::endl;

	if (!findDevicePath(wrapper->devicePath)) {
		std::cerr << "MatrixController: No AniMe Matrix Device found" << std::endl;
		return NULL;
	}

	std::cout << "MatrixController: Attempting to create file Handle to AniMe Matrix Device" << std::endl;

	wrapper->DeviceHandle = CreateFileA(wrapper->devicePath,
		GENERIC_WRITE | GENERIC_READ,
		FILE_SHARE_WRITE | FILE_SHARE_READ,
		NULL,
		OPEN_EXISTING,
		FILE_ATTRIBUTE_NORMAL | FILE_FLAG_OVERLAPPED,
		NULL
	);

	if (INVALID_HANDLE_VALUE == wrapper->DeviceHandle) {
		std::cerr << "MatrixController: Unable to create file Handle to AniMe Matrix Device: " << GetLastError() << std::endl;
		free(wrapper);
		return NULL;
	}

	std::cout << "MatrixController: Attempting to initialize WinUSB Handle" << std::endl;

	if (!WinUsb_Initialize(wrapper->DeviceHandle, &wrapper->WinusbHandle)) {
		std::cerr << "MatrixController: Unable to initialize WinUSB Handle: " << GetLastError() << std::endl;
		CloseHandle(wrapper->DeviceHandle);
		free(wrapper);
		return NULL;
	}

	std::cout << "MatrixController: Attempting to initialize AniMe Matrix Device" << std::endl;

	MatrixController* mc = new MatrixController(wrapper->WinusbHandle);
	if (!mc->initMatrix()) {
		std::cerr << "MatrixController: Unable to initialize AniMe Matrix Device: " << GetLastError() << std::endl;
		WinUsb_Free(wrapper->WinusbHandle);
		CloseHandle(wrapper->DeviceHandle);
		free(wrapper);
		return NULL;
	}

	wrapper->mc = mc;

	return wrapper;
}

void fnDeleteController(PAPI_WRAPPER w) {
	if (!w || !w->mc) {
		return;
	}

	WinUsb_Free(w->WinusbHandle);
	CloseHandle(w->DeviceHandle);
	delete w->mc;
	free(w);
}

int fnPrepareDraw(PAPI_WRAPPER w, unsigned char* m, size_t len) {
	if (!w || !w->mc) {
		return static_cast<int>(MatrixStatus::NO_CONTROLLER);
	}

	if (len != 1815) {
		// Have to send a contiguous []byte in. 55 * 33 = 1815
		return static_cast<int>(MatrixStatus::MATRIX_VECTOR_INPUT_ERROR);
	}

	auto inputMatrix = w->mc->makeVector(m, len);
	MatrixStatus ret = w->mc->fillDrawBuffer(*inputMatrix);
	delete inputMatrix;

	return static_cast<int>(ret);
}

int fnDrawMatrix(PAPI_WRAPPER w) {
	if (!w || !w->mc) {
		return static_cast<int>(MatrixStatus::NO_CONTROLLER);
	}

	MatrixStatus ret = w->mc->drawMatrix();
	return static_cast<int>(ret);
}

int fnClearMatrix(PAPI_WRAPPER w) {
	if (!w || !w->mc) {
		return static_cast<int>(MatrixStatus::NO_CONTROLLER);
	}

	MatrixStatus ret = w->mc->clearMatrix();
	return static_cast<int>(ret);
}