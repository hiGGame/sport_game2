package errs

const (
	CodeOK = 0

	CodeInternal = 50000

	CodeDBError    = 50001
	CodeValidation = 50002
	CodeNotFound   = 50003
	CodeMatchStopped  = 50004
	CodeMatchLocked   = 50005
	CodeMatchStarted  = 50006
	CodeMatchFinished = 50007
	CodeDuplicateBet  = 50008
	CodeInsufficientFunds = 50009
	CodeWechatError = 50010
)
