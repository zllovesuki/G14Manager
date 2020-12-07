#include "pch.hxx"
#include "api.hxx"
#include "binding.h"

#ifdef __cplusplus
extern "C" {
#endif
	void* NewController() {
		return fnNewController();
	}

	void DeleteController(void* w) {
		fnDeleteController(reinterpret_cast<PAPI_WRAPPER>(w));
	}

	int PrepareDraw(void* w, unsigned char* m, size_t len) {
		return fnPrepareDraw(reinterpret_cast<PAPI_WRAPPER>(w), m, len);
	}

	int DrawMatrix(void* w) {
		return fnDrawMatrix(reinterpret_cast<PAPI_WRAPPER>(w));
	}

	int ClearMatrix(void* w) {
		return fnClearMatrix(reinterpret_cast<PAPI_WRAPPER>(w));
	}
#ifdef __cplusplus
}
#endif