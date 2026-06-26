//go:build darwin

package companion

// macOS companion mode is not implemented yet. Return nil so the CLI uses standalone auth.
func platformProvider() Provider {
	return nil
}
