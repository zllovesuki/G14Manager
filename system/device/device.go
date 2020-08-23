package device

import (
	"errors"
	"log"

	"golang.org/x/sys/windows"
)

type Control struct {
	path        string
	handle      windows.Handle
	controlCode uint32
}

func NewControl(path string, controlCode uint32) (*Control, error) {
	if len(path) == 0 {
		return nil, errors.New("path cannot be empty")
	}
	h, err := windows.CreateFile(
		windows.StringToUTF16Ptr(path),
		// 0x80 is FILE_READ_ATTRIBUTES https://docs.microsoft.com/en-us/windows/win32/fileio/file-access-rights-constants
		0x80|windows.GENERIC_READ|windows.GENERIC_WRITE|windows.SYNCHRONIZE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		// FILE_NON_DIRECTORY_FILE | FILE_SYNCHRONOUS_IO_NONALERT https://processhacker.sourceforge.io/doc/ntioapi_8h.html
		0x00000040|0x00000020,
		0,
	)
	if err != nil {
		return nil, err
	}

	return &Control{
		path:        path,
		handle:      h,
		controlCode: controlCode,
	}, nil
}

func (d *Control) Write(input []byte) (int, error) {
	outBuf := make([]byte, 1024)
	outBufWritten := uint32(0)
	log.Printf("device: %s (%d) write input buffer: %+v\n", d.path, d.controlCode, input)
	err := windows.DeviceIoControl(
		d.handle,
		d.controlCode,
		&input[0],
		uint32(len(input)),
		&outBuf[0],
		uint32(len(outBuf)),
		&outBufWritten,
		nil,
	)
	if err != nil {
		return 0, err
	}
	log.Printf("device: write output buffer [0:4]: +%v\n", outBuf[0:4])
	return len(input), nil
}

func (d *Control) Read(outBuf []byte) (int, error) {
	outBufWritten := uint32(0)
	log.Printf("device: %s (%d) read input buffer: %+v\n", d.path, d.controlCode, outBuf)
	err := windows.DeviceIoControl(
		d.handle,
		d.controlCode,
		nil,
		0,
		&outBuf[0],
		uint32(len(outBuf)),
		&outBufWritten,
		nil,
	)
	if err != nil {
		return 0, err
	}
	return int(outBufWritten), nil
}

func (d *Control) Close() error {
	return windows.CloseHandle(d.handle)
}
