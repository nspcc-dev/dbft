package consensus

func secToNanoSec(s uint32) uint64 {
	return uint64(s) * 1000000000
}

func nanoSecToSec(ns uint64) uint32 {
	return uint32(ns / 1000000000)
}
