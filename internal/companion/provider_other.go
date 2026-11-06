//go:build !windows && !darwin && !linux

package companion

func platformProvider() Provider {
	return nil
}
