package gpu

type gpuInvalidStateError struct {
	msg string
}

var _ error = &gpuInvalidStateError{}

func (g gpuInvalidStateError) Error() string {
	return g.msg
}

type gpuGeneralError struct {
	msg string
}

var _ error = &gpuGeneralError{}

func (g gpuGeneralError) Error() string {
	return g.msg
}
