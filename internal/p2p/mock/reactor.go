package mock

import (
	"github.com/providenetwork/tendermint/internal/p2p"
	"github.com/providenetwork/tendermint/internal/p2p/conn"
	"github.com/providenetwork/tendermint/libs/log"
)

type Reactor struct {
	p2p.BaseReactor
}

func NewReactor() *Reactor {
	r := &Reactor{}
	r.BaseReactor = *p2p.NewBaseReactor("Mock-PEX", r)
	r.SetLogger(log.TestingLogger())
	return r
}

func (r *Reactor) GetChannels() []*conn.ChannelDescriptor            { return []*conn.ChannelDescriptor{} }
func (r *Reactor) AddPeer(peer p2p.Peer)                             {}
func (r *Reactor) RemovePeer(peer p2p.Peer, reason interface{})      {}
func (r *Reactor) Receive(chID byte, peer p2p.Peer, msgBytes []byte) {}
