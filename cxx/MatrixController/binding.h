#pragma once

#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

    void* NewController(void);
    void DeleteController(void* w);

    int PrepareDraw(void* w, unsigned char *m, size_t len);
    int DrawMatrix(void* w);
    int ClearMatrix(void* w);

#ifdef __cplusplus
}
#endif