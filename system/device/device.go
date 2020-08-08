package device

import (
	"errors"

	"golang.org/x/sys/windows"
)

type Control struct {
	handle      windows.Handle
	controlCode uint32
}

type DeviceOutput struct {
	Buffer  []byte
	Written uint32
}

func NewControl(path string, controlCode uint32) (*Control, error) {
	if len(path) == 0 {
		return nil, errors.New("path cannot be empty")
	}
	h, err := windows.CreateFile(
		windows.StringToUTF16Ptr(path),
		0xc0000000, // GENERIC_READ_AND_WRITE https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpepnp/0f2fef8b-ef79-40ff-8314-5f56787e6d9d
		3,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_DEVICE,
		0,
	)
	if err != nil {
		return nil, err
	}

	return &Control{
		handle:      h,
		controlCode: controlCode,
	}, nil
}

func (d *Control) Write(input []byte) (*DeviceOutput, error) {
	outBuf := make([]byte, 1024)
	outBufWritten := uint32(0)
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
		return nil, windows.GetLastError()
	}
	return &DeviceOutput{
		Buffer:  outBuf,
		Written: outBufWritten,
	}, nil
}

func (d *Control) Read(outputBufferLength int) (*DeviceOutput, error) {
	outBuf := make([]byte, outputBufferLength)
	outBufWritten := uint32(0)
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
		return nil, windows.GetLastError()
	}
	return &DeviceOutput{
		Buffer:  outBuf,
		Written: outBufWritten,
	}, nil
}

func (d *Control) Close() error {
	return windows.CloseHandle(d.handle)
}
