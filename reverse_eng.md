### ATKACPI (Control Code: uint32(2237452))

| Purpose              | InBuffer                                            | `%x`                      | InBufferSize | OutBuffer      | OutBufferSize | BytesReturned | Overlapped |
|----------------------|-----------------------------------------------------|---------------------------|--------------|----------------|---------------|---------------|------------|
| Battery Charge Limit | `[44 45 56 53 08 00 00 00 57 00 12 00 %x 00 00 00]` | Battery Charge Percentage | 16           | First byte = 1 | 1024          | 1024          | NULL       |