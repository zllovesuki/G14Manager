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