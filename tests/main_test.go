package tests

import (
	"testing"
)

// TestAdd prüft die Additionsfunktion.
func TestMain(t *testing.T) {
	// Die Utils Funktionen werden getestet
	t.Log("Utils testing...")
	utilsTesting(t)

	// Der Finale Socket Test wirdd durchgeführt
	t.Log("Socket testing...")
	socketTesting(t)
}
