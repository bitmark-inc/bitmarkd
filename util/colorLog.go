package util

import "github.com/bitmark-inc/logger"

const (
	//CoReset ANSI Color Code
	CoReset      = "\x1b[0m"
	CoBright     = "\x1b[1m"
	CoDim        = "\x1b[2m"
	CoUnderscore = "\x1b[4m"
	Cblink       = "\x1b[5m"
	CoReverse    = "\x1b[7m"
	CoHidden     = "\x1b[8m"
	//Colors
	CoBlack   = "\x1b[30m"
	CoRed     = "\x1b[31m"
	CoGreen   = "\x1b[32m"
	CoYellow  = "\x1b[33m"
	CoBlue    = "\x1b[34m"
	CoMagenta = "\x1b[35m"
	CoCyan    = "\x1b[36m"
	CoWhite   = "\x1b[37m"
	//Light Colors
	CoLightGray    = "\x1b[90m"
	CoLightRed     = "\x1b[91m"
	CoLightGreen   = "\x1b[92m"
	CoLightYellow  = "\x1b[93m"
	CoLightBlue    = "\x1b[94m"
	CoLightMagenta = "\x1b[95m"
	CoLightGyan    = "\x1b[96m"
	CoLightWhite   = "\x1b[97m"

	//Background Color
	CoBGblack   = "\x1b[40m"
	CoBGred     = "\x1b[41m"
	CoBGgreen   = "\x1b[42m"
	CoBGyellow  = "\x1b[43m"
	CoBGblue    = "\x1b[44m"
	CoBGmagenta = "\x1b[45m"
	CoBGcyan    = "\x1b[46m"
	CoBGwhite   = "\x1b[47m"
)

//LogDebug print  message in Debug level with assigned color
func LogDebug(log *logger.L, color string, message string) {
	log.Debugf("%s%s%s", color, message, CoReset)
}

//LogInfo print  message in Info level with assigned color
func LogInfo(log *logger.L, color string, message string) {
	log.Infof("%s%s%s", color, message, CoReset)
}

//LogError print  message in Error level with assigned color
func LogError(log *logger.L, color string, message string) {
	log.Errorf("%s%s%s", color, message, CoReset)
}

//LogWarn print  message in Warn level with assigned color
func LogWarn(log *logger.L, color string, message string) {
	log.Warnf("%s%s%s", color, message, CoReset)
}
