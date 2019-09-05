package cluster

var zero = struct{}{}

type Broadcaster interface {
	Register(chan<- interface{})
	UnRegister(chan<- interface{})
	Close()
	Submit(interface{})
}

type broadcaster struct {
	input chan interface{}

	reg   chan chan<- interface{}
	unReg chan chan<- interface{}

	subscribers map[chan<- interface{}]struct{}
}

func NewBroadcaster(bufLen int, stop chan<- struct{}) Broadcaster {
	inst := &broadcaster{
		input:       make(chan interface{}, bufLen),
		reg:         make(chan chan<- interface{}),
		unReg:       make(chan chan<- interface{}),
		subscribers: make(map[chan<- interface{}]struct{}),
	}
	go inst.run(stop)
	return inst
}

func (b *broadcaster) broadcast(d interface{}) {
	for ch := range b.subscribers {
		ch <- d
	}
}

func (b *broadcaster) run(stop chan<- struct{}) {
	for {
		select {
		case d := <-b.input:
			b.broadcast(d)
		case ch, ok := <-b.reg:
			if ok {
				b.subscribers[ch] = zero
			}
		case ch := <-b.unReg:
			delete(b.subscribers, ch)
		}
	}
}

func (b *broadcaster) Register(dataChan chan<- interface{}) {
	b.reg <- dataChan
}

func (b *broadcaster) UnRegister(dataChan chan<- interface{}) {
	b.unReg <- dataChan
}

func (b *broadcaster) Close() {
	close(b.reg)
}

func (b *broadcaster) Submit(data interface{}) {
	if b != nil {
		b.input <- data
	}
}
