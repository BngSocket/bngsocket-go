package bngsocket

import "sync"

func newConnACK() *_ConnACK {
	n := new(_ConnACK)
	n.mutex = new(sync.Mutex)
	n.cond = sync.NewCond(n.mutex)
	n.state = 0
	return n
}

func (n *_ConnACK) WaitOfACK() error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	// Wait until state is not zero
	for n.state == 0 {
		n.cond.Wait()
	}

	// Der Status wird auf 0 gesetzt
	n.state = 0

	return nil
}

func (n *_ConnACK) EnterACK() error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	// Set state to indicate that ACK has been entered
	n.state = 1

	// Signal all waiting goroutines
	n.cond.Broadcast()
	return nil
}
