#ifdef __cplusplus
extern "C"
{
#endif

    void *fnGetDisplay(void);
    int fnCycleRefreshRate(void *);
    int fnGetCurrentRefreshRate(void *);
    void fnReleaseDisplay(void *);

#ifdef __cplusplus
}
#endif