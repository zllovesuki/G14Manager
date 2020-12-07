// The following ifdef block is the standard way of creating macros which make exporting
// from a DLL simpler. All files within this DLL are compiled with the MATRIXCONTROLLER_EXPORTS
// symbol defined on the command line. This symbol should not be defined on any project
// that uses this DLL. This way any other project whose source files include this file see
// MATRIXCONTROLLER_API functions as being imported from a DLL, whereas this DLL sees symbols
// defined with this macro as being exported.
// An additional symbol GO_BINDINGS is check for cgo linker. If GO_BINDINGS is defined, then
// no dllexport/dllimport will be compiled.

#ifdef MATRIXCONTROLLER_EXPORTS
#define MATRIXCONTROLLER_API __declspec(dllexport)
#else
#ifdef GO_BINDINGS
#define MATRIXCONTROLLER_API
#else
#define MATRIXCONTROLLER_API __declspec(dllimport)
#endif
#endif


#include "MatrixController.hxx"

typedef struct _API_WRAPPER {
	WINUSB_INTERFACE_HANDLE WinusbHandle;
	HANDLE                  DeviceHandle;
	MatrixController		*mc;
	char					devicePath[MAX_PATH + 1];
} API_WRAPPER, *PAPI_WRAPPER;

#ifdef __cplusplus
extern "C" {
#endif

	MATRIXCONTROLLER_API PAPI_WRAPPER fnNewController(void);
	MATRIXCONTROLLER_API void fnDeleteController(PAPI_WRAPPER w);

	MATRIXCONTROLLER_API int fnPrepareDraw(PAPI_WRAPPER w, unsigned char* m, size_t len);
	MATRIXCONTROLLER_API int fnDrawMatrix(PAPI_WRAPPER w);
	MATRIXCONTROLLER_API int fnClearMatrix(PAPI_WRAPPER w);

#ifdef __cplusplus
}
#endif
