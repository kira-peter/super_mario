//go:build !windows

package app

// platformProbe returns nil off Windows: there is no portable way to read the
// physical keyboard from a terminal program, so key state stays inferred from
// the presses the terminal delivers.
func platformProbe() keyProbe { return nil }
