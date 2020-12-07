#include "pch.hxx"

constexpr auto MATRIXHEIGHT = 55;

typedef std::vector<std::vector<BYTE>> ByteVec2D;

class MatrixController {
private:
	const static USHORT PACKET_LENGTH = 640;
	const static BYTE DEVICE_PAGE = 0x5e;
	constexpr const static unsigned char INIT_STR[15] = "ASUS Tech.Inc.";

	constexpr const static int ROW_WIDTH[55] = {
		33,33,33,33,33,
		33,33,32,32,31,
		31,30,30,29,29,
		28,28,27,27,26,
		26,25,25,24,24,
		23,23,22,22,21,
		21,20,20,19,19,
		18,18,17,17,16,
		16,15,15,14,14,
		13,13,12,12,11,
		11,10,10,9,9
	};
	constexpr const static int ROW_INDEX[MATRIXHEIGHT][2] = {
		{7, 39},
		{41, 73},
		{76, 108},
		{109, 141},
		{144, 176},
		{177, 209},
		{211, 243},
		{244, 275},
		{277, 308},
		{309, 339},
		{341, 371},
		{372, 401},
		{403, 432},
		{433, 461},
		{463, 491},
		{492, 519},
		{521, 548},
		{549, 575},
		{577, 603},
		{604, 629},
		{631, 656},
		{30, 54},
		{56, 80},
		{81, 104},
		{106, 129},
		{130, 152},
		{154, 176},
		{177, 198},
		{200, 221},
		{222, 242},
		{244, 264},
		{265, 284},
		{286, 305},
		{306, 324},
		{326, 344},
		{345, 362},
		{364, 381},
		{382, 398},
		{400, 416},
		{417, 432},
		{434, 449},
		{450, 464},
		{466, 480},
		{481, 494},
		{496, 509},
		{510, 522},
		{524, 536},
		{537, 548},
		{550, 561},
		{562, 572},
		{574, 584},
		{585, 594},
		{596, 605},
		{606, 614},
		{616, 624},
	};
	constexpr const static BYTE paneHeaders[2][7] = {
		{0x5e, 0xc0, 0x02, 0x01, 0x00, 0x73, 0x02},
		{0x5e, 0xc0, 0x02, 0x74, 0x02, 0x73, 0x02}
	};

	bool hasHandle;
	WINUSB_INTERFACE_HANDLE usbHandle;
	WINUSB_SETUP_PACKET SetupPacket;
	
	BYTE firstPane[640];
	BYTE secondPane[640];
	BYTE flushPacket[640];

	std::vector<int> sizes;
	int rowSize[MATRIXHEIGHT];

public:
	inline const static std::string MATRIX_DEVICE_VEND_PROD = "VID_0B05&PID_193B";

	MatrixController(void);
	MatrixController(WINUSB_INTERFACE_HANDLE h);
	void setHandle(WINUSB_INTERFACE_HANDLE h);
	bool initMatrix();
	ByteVec2D* makeVector(unsigned char* m, size_t len);
	MatrixStatus fillDrawBuffer(ByteVec2D arr);
	MatrixStatus drawMatrix();
	MatrixStatus clearMatrix();

};