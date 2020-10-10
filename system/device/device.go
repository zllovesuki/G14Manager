package device

import (
	"errors"
	"log"

	"golang.org/x/sys/windows"
)

type Config struct {
	DryRun      bool
	Path        string
	ControlCode uint32
}

type Control struct {
	Config
	handle windows.Handle
}

func NewControl(conf Config) (*Control, error) {
	if len(conf.Path) == 0 {
		return nil, errors.New("path cannot be empty")
	}
	if conf.ControlCode == 0 {
		return nil, errors.New("control code cannot be 0")
	}
	h, err := windows.CreateFile(
		windows.StringToUTF16Ptr(conf.Path),
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
		Config: conf,
		handle: h,
	}, nil
}

func (d *Control) Write(input []byte) (int, error) {
	if d.Config.DryRun {
		log.Printf("[dry run] device: %s (%d) write input buffer [0:8]: %+v\n", d.Config.Path, d.Config.ControlCode, input[0:8])
		return len(input), nil
	}
	outBuf := make([]byte, 1024)
	outBufWritten := uint32(0)
	log.Printf("device: %s (%d) write input buffer [0:8]: %+v\n", d.Config.Path, d.Config.ControlCode, input[0:8])
	err := windows.DeviceIoControl(
		d.handle,
		d.Config.ControlCode,
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
	log.Printf("device: write output buffer [0:8]: %+v\n", outBuf[0:8])
	return len(input), nil
}

func (d *Control) Read(outBuf []byte) (int, error) {
	if d.Config.DryRun {
		log.Printf("[dry run] device: %s (%d) read input buffer [0:8]: %+v\n", d.Config.Path, d.Config.ControlCode, outBuf[0:8])
		return 0, nil
	}
	outBufWritten := uint32(0)
	log.Printf("device: %s (%d) read input buffer [0:8]: %+v\n", d.Config.Path, d.Config.ControlCode, outBuf[0:8])
	err := windows.DeviceIoControl(
		d.handle,
		d.Config.ControlCode,
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

func (d *Control) Execute(input []byte, outLen int) ([]byte, error) {
	if d.Config.DryRun {
		log.Printf("[dry run] device: %s (%d) execute input buffer [0:8]: %+v\n", d.Config.Path, d.Config.ControlCode, input[0:8])
		return make([]byte, outLen), nil
	}
	outBuf := make([]byte, 1024)
	outBufWritten := uint32(0)
	log.Printf("device: %s (%d) execute input buffer: %+v\n", d.Config.Path, d.Config.ControlCode, input)
	err := windows.DeviceIoControl(
		d.handle,
		d.Config.ControlCode,
		&input[0],
		uint32(len(input)),
		&outBuf[0],
		uint32(len(outBuf)),
		&outBufWritten,
		nil,
	)
	if err != nil {
		return nil, err
	}
	return outBuf[0:outLen], nil
}

func (d *Control) Close() error {
	return windows.CloseHandle(d.handle)
}
