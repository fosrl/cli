//go:build !windows && !darwin

package companion

func platformProvider() Provider {
	return nil
}
