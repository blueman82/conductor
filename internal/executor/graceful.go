package executor

// graceful.go provides helpers for graceful degradation patterns across hooks.
// The pattern: warn about errors but don't fail execution.

// GracefulWarn logs a warning if logger is non-nil, using the given format and args.
// This helper eliminates the repeated pattern of:
//
//	if logger != nil {
//	    logger.Warnf(format, args...)
//	}
//
// Usage:
//
//	if err != nil {
//	    GracefulWarn(h.logger, "Setup: Introspection failed: %v", err)
//	    return nil
//	}
func GracefulWarn(logger RuntimeEnforcementLogger, format string, args ...interface{}) {
	if logger != nil {
		logger.Warnf(format, args...)
	}
}

// GracefulInfo logs an info message if logger is non-nil.
// Companion to GracefulWarn for consistent logger nil-checking.
func GracefulInfo(logger RuntimeEnforcementLogger, format string, args ...interface{}) {
	if logger != nil {
		logger.Infof(format, args...)
	}
}
