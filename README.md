# ROGManager: An open source replacement to manage your G14

![Test and Build](https://github.com/zllovesuki/ROGManager/workflows/Test%20and%20Build/badge.svg) ![Build Release](https://github.com/zllovesuki/ROGManager/workflows/Build%20Release/badge.svg)

## Disclaimer

Your warranty is now void. Proceed at your own risk.

## Current Status

Follow project status on [Sprint Board](https://github.com/zllovesuki/ROGManager/projects/1). This will include features in progress.

After some reverse engineering, ROGManager now (mostly) replaces Asus software suite's functionalities. Unimplemented functionalities are:
1. ~~Toggle mute/unmute microphone~~
2. ~~Toggle enable/disable TouchPad~~
3. ~~Keyboard brightness adjustment~~
4. On-screen display
5. AniMe Matrix control

_Note_: Currently, the default profiles expect Power Plans "High Performance" and "Power Saver" to be available. If your installation of Windows does not have those Power Plans, ROGManager will refuse to start. This will be fixed when customizable config is released.

## Bug Report

If your encounter an issue with using ROGManager (e.g. does not start, functionalities not working, etc), please download the debug build `ROGManager.debug.exe`, and run the binary in a Terminal with Administrator Privileges, then submit an issue with the full logs.

## Requirements

You must have a Zephyrus G14 (GA401), and has Asus Optimization the driver (aka Asus System Control Interface V2) installed. You may check and see if `C:\Windows\System32\DriverStore\FileRepository\asussci2.inf_amd64_xxxxxxxxxxxxxxxx` exists. ROGManager is **not tested** on other Zephyrus laptops (e.g. G15). You are welcome to test ROGManager on laptops such as G15, however support will be limited, and all functionalities are not guaranteed to be working.

Asus Optimization (the service) **cannot** be running, otherwise ROGManager and Asus Optimization will be fighting over control. We only need Asus Optimization (the driver) to be installed so Windows will load `atkwmiacpi64.sys`, and exposes a `\\.\ATKACPI` device to be used.

You do not need any other softwares from Asus (e.g. Armoury Crate and its cousins, etc) running to use ROGManager; you can safely uninstall them from your system. However, some softwares (e.g. Asus Optimization) are installed as Windows Services, and you should disable them in Services as they do not provide any value:

![Running Services](images/services.png)

Optionally, disable ASUS System Analysis Driver with `sc.exe config "ASUSSAIO" start=disabled` in a Terminal with Administrator privileges, if you do not plan to use MyASUS.

Recommend running `ROGManager.exe` on startup in Task Scheduler with "Run with highest privileges."

## Remapping the ROG Key

Use case: You can compile your `.ahk` to `.exe` and run your macros.

By default, it will launch Task Manager when you press the ROG Key once.

To specify which program to launch when pressed multiple times, pass your path to the desired program as argument to `-rog` multiple times. For example:

```
.\ROGManager.exe -rog "Taskmgr.exe" -rog "start Spotify.exe"
```

This will launch Task Manager when you press the ROG key once, and Spotify when you press twice.

## Changing the Fan Curve

For the initial release, you have to change fan curve in `system\thermal\default.go`. In a future release ROGManager will allow you to specify the fan curve without rebuilding the binary. However, the default fan curve should be sufficient for most users.

Use the `Fn + F5` key combo to cycle through all the profiles. Fanless -> Quiet -> Slient -> Performance.

The key combo has a time delay. If you press the combo X times, it will apply the the next X profile. For example, if you are currently on "Fanless" profile, pressing `Fn + F5` twice will apply the "Slient" profile.

## How to Build

1. Install golang 1.14+ if you don't have it already
2. Install mingw x86_64 for `gcc.exe`
2. Install `rsrc`: `go get github.com/akavel/rsrc`
3. Generate `syso` file: `\path\to\rsrc.exe -arch amd64 -manifest ROGManager.exe.manifest -ico go.ico -o ROGManager.exe.syso`
4. Build the binary: `.\scripts\build.ps1`

## Developing

Use `.\scripts\run.ps1`.

Most keycodes can be found in [reverse_eng/codes.txt](reverse_eng/codes.txt), and the `reverse_eng` folder contains USB and API calls captures for reference.

## References:

- https://github.com/torvalds/linux/blob/master/drivers/platform/x86/asus-wmi.c
- https://github.com/torvalds/linux/blob/master/drivers/platform/x86/asus-nb-wmi.c
- https://github.com/torvalds/linux/blob/master/drivers/hid/hid-asus.c
- https://github.com/flukejones/rog-core/blob/master/kernel-patch/0001-HID-asus-add-support-for-ASUS-N-Key-keyboard-v5.8.patch
- [Reverse Engineering](./reverse_eng.md)