// MatrixController.cpp : Implementations of AniMe Matrix Control
//

#include "pch.hxx"
#include "MatrixController.hxx"

MatrixController::MatrixController(void)
{
	hasHandle = false;
	usbHandle = NULL;
	ZeroMemory(flushPacket, sizeof(BYTE) * 640);
	flushPacket[0] = DEVICE_PAGE;
	flushPacket[1] = 0xc0;
	flushPacket[2] = 0x03;
	ZeroMemory(firstPane, sizeof(BYTE) * 640);
	ZeroMemory(secondPane, sizeof(BYTE) * 640);
	ZeroMemory(&SetupPacket, sizeof(WINUSB_SETUP_PACKET));
	SetupPacket.RequestType = 0x21;
	SetupPacket.Request = 0x09;
	SetupPacket.Value = 0x035e;
	SetupPacket.Index = 0x00;
	SetupPacket.Length = PACKET_LENGTH;

	for (int i = 0; i < MATRIXHEIGHT; i++)
	{
		rowSize[i] = ROW_INDEX[i][1] - ROW_INDEX[i][0] + 1;
	}

	return;
}

MatrixController::MatrixController(WINUSB_INTERFACE_HANDLE h) : MatrixController() {
	setHandle(h);
}

void MatrixController::setHandle(WINUSB_INTERFACE_HANDLE h) {
	hasHandle = true;
	usbHandle = h;
}

bool MatrixController::initMatrix() {
	if (!hasHandle)
	{
		return false;
	}

	ULONG cbSent = 0;
	BYTE packet[640];
	ZeroMemory(packet, sizeof(BYTE) * 640);
	packet[0] = DEVICE_PAGE;
	for (int i = 0; i < 14; i++)
	{
		packet[i + 1] = INIT_STR[i];
	}

	if (!WinUsb_ControlTransfer(usbHandle, SetupPacket, packet, 640, &cbSent, 0) || cbSent != 640)
	{
		return false;
	}
	return true;
}

ByteVec2D* MatrixController::makeVector(unsigned char* m, size_t len) {
	ByteVec2D* inputMatrix = new ByteVec2D();
	for (int i = 0; i < MATRIXHEIGHT; i++)
	{
		std::vector<BYTE> temp;
		int colOffset = i * MATRIXHEIGHT;

		for (int j = 0; j < ROW_WIDTH[i]; j++)
		{
			temp.push_back(*(m + colOffset + j));
		}
		inputMatrix->push_back(temp);
	}
	return inputMatrix;
}

MatrixStatus MatrixController::fillDrawBuffer(ByteVec2D arr) {
	if (arr.size() != MATRIXHEIGHT)
	{
		return MatrixStatus::MATRIX_VECTOR_INPUT_ERROR;
	}

	ZeroMemory(firstPane, sizeof(BYTE) * 640);
	ZeroMemory(secondPane, sizeof(BYTE) * 640);
	for (int i = 0; i < 7; i++)
	{
		firstPane[i] = paneHeaders[0][i];
		secondPane[i] = paneHeaders[1][i];
	}
	for (int i = 0; i < MATRIXHEIGHT; i++)
	{
		if (arr[i].size() != rowSize[i])
		{
			return MatrixStatus::MATRIX_VECTOR_INPUT_ERROR;
		}
		else
		{
			if (i < 20)
			{
				if (i == 0)
				{
					for (size_t j = 0; j < arr[i].size() - 1; j++)
					{
						firstPane[ROW_INDEX[i][1] - j] = arr[i][j + 1];
					}
				}
				else
				{
					for (size_t j = 0; j < arr[i].size(); j++)
					{
						firstPane[ROW_INDEX[i][1] - j] = arr[i][j];
					}
				}
			}
			else if (i > 20)
			{
				for (size_t j = 0; j < arr[i].size(); j++)
				{
					secondPane[ROW_INDEX[i][1] - j] = arr[i][j];
				}
			}
			else if (i == 20)
			{
				for (int j = 0; j < 23; j++)
				{
					secondPane[29 - j] = arr[i][j];
				}
				for (int j = 23; j < 26; j++)
				{
					firstPane[ROW_INDEX[i][1] - j] = arr[i][j];
				}
			}
		}
	}

	return MatrixStatus::MATRIX_OPERATION_SUCCESSFUL;
}

MatrixStatus MatrixController::drawMatrix() {
	if (!hasHandle)
	{
		return MatrixStatus::MATRIX_NO_HANDLE;
	}

	ULONG bytesSent = 0;
	if (!WinUsb_ControlTransfer(usbHandle, SetupPacket, firstPane, 640, &bytesSent, 0) || bytesSent != 640)
	{
		return MatrixStatus::MATRIX_DRAW_ERROR;
	}
	if (!WinUsb_ControlTransfer(usbHandle, SetupPacket, secondPane, 640, &bytesSent, 0) || bytesSent != 640)
	{
		return MatrixStatus::MATRIX_DRAW_ERROR;
	}
	if (!WinUsb_ControlTransfer(usbHandle, SetupPacket, flushPacket, 640, &bytesSent, 0) || bytesSent != 640)
	{
		return MatrixStatus::MATRIX_DRAW_ERROR;
	}
	return MatrixStatus::MATRIX_OPERATION_SUCCESSFUL;
}

MatrixStatus MatrixController::clearMatrix() {
	if (!hasHandle)
	{
		return MatrixStatus::MATRIX_NO_HANDLE;
	}

	ByteVec2D inputMatrix;
	for (int i = 0; i < MATRIXHEIGHT; i++)
	{
		std::vector<BYTE> temp;
		for (int j = 0; j < ROW_WIDTH[i]; j++)
		{
			temp.push_back(0x00);
		}
		inputMatrix.push_back(temp);
	}
	fillDrawBuffer(inputMatrix);
	return drawMatrix();
}
