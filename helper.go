package bngsocket

// Gibt an ob die Bng Verbindung geschlossen wurde
func IsConnectionClosed(conn *BngConn) bool {
	err := conn.runningError.Get()
	if err != nil {
		return true
	}

	if conn.closed.Get() {
		return true
	}

	if conn.closing.Get() {
		return true
	}

	return false
}

// Wartet darauf dass sich der Status einer Bng Verbindung ändert
func WaitOfConnectionStateChange(conn *BngConn) bool {
	return false
}
