package netio

import (
	"container/list"
	"sync"
	"time"
)
type MovingAverage struct {
	sync.Mutex
	period int
	stream *list.List
	sum int64
	accumulator int64
	lastAverage float64
}
func NewMovingAverage(period int) *MovingAverage {
	if period == 0 {
		period = 10
	}
	return &MovingAverage{
		period : period,
		stream : list.New(),
	}
}
func (self *MovingAverage) add(n int64) {
	self.Lock()
	defer self.Unlock()
	self.accumulator += n
}
func (self *MovingAverage) flush() {
	self.Lock()
	defer self.Unlock()
	
	n := self.accumulator
	self.accumulator = 0
	stream := self.stream
	stream.PushBack(n)
	self.sum += n
	
	streamlen := stream.Len()
	
	if streamlen > self.period {
		ele := stream.Front()
		stream.Remove(ele)
		dn := ele.Value.(int64)
		self.sum -= dn
		streamlen -= 1
	}
	if streamlen == 0 {
		self.lastAverage = 0
	}else{
		self.lastAverage = float64(self.sum) / float64(streamlen)
	}
}

type StatsResult struct {
	StartTime time.Time
	MaxSession int64
	ActiveSession int64
	
	MaxConnections float64
	ActiveConnections float64
	
	ConnectionsPs float64
	
	PacketsSentPs float64
	PacketsRecvPs float64
}

type  StatsCollector struct {
	mutex sync.Mutex
	
	StartTime time.Time
	
	MaxSession int64
	ActiveSession int64
	
	MaxConnections float64
	ActiveConnections float64
	ConnectionsPs *MovingAverage
	
	PacketsSentPs *MovingAverage
	PacketsRecvPs *MovingAverage
}

func NewStatsCollector() *StatsCollector{
	return &StatsCollector{
		ConnectionsPs: NewMovingAverage(0),
		PacketsRecvPs : NewMovingAverage(0),
		PacketsSentPs: NewMovingAverage(0),
	}
}

func (s *StatsCollector) SessionOpened() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.ActiveSession += 1
	if s.ActiveSession > s.MaxSession {
		s.MaxSession = s.ActiveSession
	}
}

func (s *StatsCollector) SessionClosed() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.ActiveSession -= 1
}

func (s *StatsCollector) ConnectionOpened() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.ActiveConnections += 1
	if s.ActiveConnections > s.MaxConnections {
		s.MaxConnections = s.ActiveConnections
	}
	s.ConnectionsPs.add(1)
}
func (s *StatsCollector) ConnectionClosed() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.ActiveConnections -= 1
}
func (s *StatsCollector) OnPacketSent(num int64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.PacketsSentPs.add(num)
}
func (s *StatsCollector) OnPacketRecv(num int64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.PacketsRecvPs.add(num)
}

func (s *StatsCollector) Dump() *StatsResult {
	return &StatsResult {
		
		StartTime : s.StartTime,
		
		ActiveSession : s.ActiveSession,
		MaxSession : s.MaxSession,
		
		ActiveConnections : s.ActiveConnections,
		MaxConnections : s.MaxConnections,
		ConnectionsPs : s.ConnectionsPs.lastAverage,
		
		PacketsRecvPs : s.PacketsRecvPs.lastAverage,
		PacketsSentPs : s.PacketsSentPs.lastAverage,
	}
}

func (s *StatsCollector) updateAverages() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.ConnectionsPs.flush()
	s.PacketsRecvPs.flush()
	s.PacketsSentPs.flush()
}
func (s* StatsCollector) Start() {
	s.StartTime = time.Now()
	go func(){
		ticker := time.NewTicker(1 * time.Second)
		for _ = range ticker.C{
			s.updateAverages()
		}
	}()
}