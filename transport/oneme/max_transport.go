package oneme

import (
	"universal-bypass-tool/transport"
	"universal-bypass-tool/utils"
)

type OneMeTransport struct {
	b     *transport.BaseTransport
	token string
	uid   int64
	exit  bool

	oneMeClient MaxClient
	ch          *CallHandler
}

func (t *OneMeTransport) Receive(callback func([]byte)) {
	t.b.Receive(callback)
}

func (t *OneMeTransport) Stats() transport.TransportStats {
	return t.b.Stats()
}

func NewOneMeTransport(isExit bool, maxToken string, maxUid int64, config transport.TransportConfig) *OneMeTransport {
	return &OneMeTransport{
		b:     transport.NewBaseTransport(config),
		token: maxToken,
		uid:   maxUid,
		exit:  isExit,
	}
}

func (t *OneMeTransport) Start() error {
	utils.Debugf("creating max client ...")
	t.oneMeClient = *NewMaxClient()
	if err := t.oneMeClient.Connect(); err != nil {
		return err
	}
	if err := t.oneMeClient.LoginByToken(t.token); err != nil {
		return err
	}

	if t.exit {
		utils.Debugf("configured ch for exit node")
		t.ch = startIncomingListener(&t.oneMeClient)
	} else {
		utils.Debugf("configured ch for client mode")
		t.ch = startOutgoingCall(&t.oneMeClient, t.uid)
	}

	utils.Debugf("configured dc inbound")
	t.ch.dcInbound = func(data []byte) {
		t.b.CallReceive(data)
	}

	t.b.SetConnected(true)
	return t.b.Start()
}

func (t *OneMeTransport) Stop() error {
	t.b.SetConnected(false)
	if t.ch != nil {
		t.ch.mu.Lock()
		if t.ch.conn != nil {
			t.ch.conn.Close()
			t.ch.conn = nil
		}
		t.ch.mu.Unlock()
	}
	return t.b.Stop()
}

func (t *OneMeTransport) IsConnected() bool {
	if t.ch == nil {
		return false
	}
	t.ch.mu.Lock()
	connAlive := t.ch.conn != nil
	t.ch.mu.Unlock()
	return t.b.IsConnected() && connAlive
}

func (t *OneMeTransport) Send(data []byte) error {
	t.ch.Send(data)
	return nil
}
