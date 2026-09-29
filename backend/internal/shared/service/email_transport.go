package service

import (
	"fmt"
	"sync"
)

var logTransportWarnOnce = sync.OnceFunc(func() {
	fmt.Println("[EMAIL WARNING] EMAIL_LOG_ONLY=true: emails are being logged to stdout and NOT sent. This is a development-only mode.")
})
