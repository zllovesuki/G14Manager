### ATKACPI (Control Code: uint32(2237452))

| Purpose              | InBuffer                                            | `%x`                      | InBufferSize | OutBuffer      | OutBufferSize | BytesReturned | Overlapped |
|----------------------|-----------------------------------------------------|---------------------------|--------------|----------------|---------------|---------------|------------|
| Battery Charge Limit | `[44 45 56 53 08 00 00 00 57 00 12 00 %x 00 00 00]` | Battery Charge Percentage | 16           | First byte = 1 | 1024          | 1024          | NULL       |

### Keyboard

ROG Key Pressed

#	Type	Name	Pre-Call Value	Post-Call Value
6	ULONG	IoControlCode	2237452	2237452
#	Type	Name	Pre-Call Value	Post-Call Value
7	PVOID	InputBuffer	0x000001cfd6dc06d0	0x000001cfd6dc06d0
`[44 45 56 53 08 00 00 00 21 00 10 00 38 00 00 00]`

#	Type	Name	Pre-Call Value	Post-Call Value
6	ULONG	IoControlCode	2237448	2237448
#	Type	Name	Pre-Call Value	Post-Call Value
9	PVOID	OutputBuffer	0x000000bcc0afe15c	0x000000bcc0afe15c
`[38 00 00 00]`


### Keyboard backlight
```
Caption                     : HID-compliant vendor-defined device
Description                 : HID-compliant vendor-defined device
InstallDate                 : 
Name                        : HID-compliant vendor-defined device
Status                      : OK
Availability                : 
ConfigManagerErrorCode      : CM_PROB_NONE
ConfigManagerUserConfig     : False
CreationClassName           : Win32_PnPEntity
DeviceID                    : HID\VID_0B05&PID_1866&MI_02&COL01\8&1E16C781&0&0000
ErrorCleared                : 
ErrorDescription            : 
LastErrorCode               : 
PNPDeviceID                 : HID\VID_0B05&PID_1866&MI_02&COL01\8&1E16C781&0&0000
PowerManagementCapabilities : 
PowerManagementSupported    : 
StatusInfo                  : 
SystemCreationClassName     : Win32_ComputerSystem
SystemName                  : RACHEL-G14
ClassGuid                   : {745a17a0-74d3-11d0-b6fe-00a0c90f57da}
CompatibleID                : 
HardwareID                  : {HID\VID_0B05&PID_1866&REV_0002&MI_02&Col01, HID\VID_0B05&PID_1866&MI_02&Col01, HID\VID_0B05&UP:FF31_U:0076, 
                              HID_DEVICE_UP:FF31_U:0076...}
Manufacturer                : (Standard system devices)
PNPClass                    : HIDClass
Present                     : True
Service                     : 
PSComputerName              : 
Class                       : HIDClass
FriendlyName                : HID-compliant vendor-defined device
InstanceId                  : HID\VID_0B05&PID_1866&MI_02&COL01\8&1E16C781&0&0000
Problem                     : CM_PROB_NONE
ProblemDescription          : 
```

#	Type	Name	Pre-Call Value	Post-Call Value
6	ULONG	IoControlCode	721297	721297
#	Type	Name	Pre-Call Value	Post-Call Value
7	PVOID	InputBuffer	0x000001b7763041d0	0x000001b7763041d0
`[0x5a, 0xba, 0xc5, 0xc4, %x, ...]` len(64) padded with 0x00
%x: 00 backlight off, 01 02 03 different levels