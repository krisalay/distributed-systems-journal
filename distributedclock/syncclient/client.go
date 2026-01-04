package syncclient

import "github.com/krisalay/distributed-systems-journal/distributedclock/hlc"

// AdjustedTime returns network-corrected server time
func AdjustedTime(serverTS hlc.Timestamp, rttMillis int64) int64 {
	return serverTS.Physical + rttMillis/2
}

// TimeLeft computes remaining exam time
func TimeLeft(endTime hlc.Timestamp, serverTS hlc.Timestamp, rttMillis int64) int64 {
	adjusted := AdjustedTime(serverTS, rttMillis)
	return endTime.Physical - adjusted
}
