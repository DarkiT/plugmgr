package eventbus

var (
	ErrChannelClosed = err{Code: 10004, Msg: "channel is closed"}
)

type err struct {
	Msg  string
	Code int
}

// String return the error's message
func (e err) String() string {
	return e.Msg
}

// Error return the error's message
func (e err) Error() string {
	return e.Msg
}
