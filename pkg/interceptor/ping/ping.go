package ping

import (
	"time"

	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

type ping struct {
	rtt       time.Duration // Round-trip time for ping-pong
	timestamp time.Time     // When this ping was recorded
}

type pings struct {
	pings  []ping // Historical pings, limited by max capacity
	max    uint16 // Maximum number of pings to keep
	recvd  int    // Total pongs received
	count  int    // Total pings sent
	recent ping   // Most recent ping
}

// recordPong updates pings based on a received pong message
func (s *pings) recordPong(msg *message.Pong) {
	rtt := msg.Timestamp.Sub(msg.PingTimestamp)

	newStat := ping{
		rtt:       rtt,
		timestamp: time.Now(),
	}

	s.recent = newStat

	if uint16(len(s.pings)) >= s.max {
		if len(s.pings) > 0 {
			s.pings = s.pings[1:]
		}
	}
	s.pings = append(s.pings, newStat)
	s.recordReceivedPong(msg)
}

// recordSentPing increments the count of pings sent
func (s *pings) recordSentPing(_ *message.Ping) {
	s.count++
}

func (s *pings) recordReceivedPong(_ *message.Pong) {
	s.recvd++
}

// GetRecentRTT returns the most recent round-trip time
func (s *pings) GetRecentRTT() time.Duration {
	return s.recent.rtt
}

// GetAverageRTT calculates the average round-trip time
func (s *pings) GetAverageRTT() time.Duration {
	if len(s.pings) == 0 {
		return 0
	}

	var total time.Duration
	for _, stat := range s.pings {
		total += stat.rtt
	}

	return total / time.Duration(len(s.pings))
}

// GetMaxRTT returns the maximum round-trip time observed
func (s *pings) GetMaxRTT() time.Duration {
	if len(s.pings) == 0 {
		return 0
	}

	var maxRTT time.Duration
	for _, stat := range s.pings {
		if stat.rtt > maxRTT {
			maxRTT = stat.rtt
		}
	}

	return maxRTT
}

// GetMinRTT returns the minimum round-trip time observed
func (s *pings) GetMinRTT() time.Duration {
	if len(s.pings) == 0 {
		return 0
	}

	minRTT := s.pings[0].rtt
	for _, stat := range s.pings {
		if stat.rtt < minRTT {
			minRTT = stat.rtt
		}
	}

	return minRTT
}

// GetSuccessRate returns the percentage of successful pings
func (s *pings) GetSuccessRate() float64 {
	if s.count == 0 {
		return 0
	}

	return 100.0 * (1.0 - float64(s.count-s.recvd)/float64(s.count))
}

type statsOption = func(*pings) error

func withMax(max uint16) statsOption {
	return func(s *pings) error {
		s.max = max
		return nil
	}
}

type statsFactory struct {
	opts []statsOption
}

func (factory *statsFactory) createStats() (*pings, error) {
	stats := &pings{
		pings:  make([]ping, 0),
		max:    ^uint16(0),
		count:  0,
		recvd:  0,
		recent: ping{},
	}

	for _, option := range factory.opts {
		if err := option(stats); err != nil {
			return nil, err
		}
	}

	return stats, nil
}
