package brain

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gillepool/botty/internal/events"
)

type Brain struct {
	eventsInput chan Event // input for any new events, the Brain ensures that callers never block when writing to it
	eventsLoop  chan Event // used in Brain.HandleEvents() to actually process the events
	shutdown    chan shutdownRequest

	mu             sync.RWMutex // mu protects concurrent access to the handlers
	handlers       map[reflect.Type][]eventHandler
	handlerTimeout time.Duration // zero means no timeout, defaults to one minute

	RegistrationErrs []error // any errors that occurred during setup (e.g. in Bot.RegisterHandler)
	handlingEvents   int32   // accessed atomically (non-zero means the event handler was started)
	closed           int32   // accessed atomically (non-zero means the brain was shutdown already)
}

type Event struct {
	Data       interface{}
	Callbacks  []func(Event)
	AbortEarly bool
}

// The shutdownRequest type is used when signaling shutdown information between
// Brain.Shutdown() and the Brain.HandleEvents loop.
type shutdownRequest struct {
	ctx      context.Context
	callback chan bool
}

// An eventHandler is a function that takes a context and the reflected value
// of a concrete event type.
type eventHandler func(context.Context, reflect.Value) error

func FinishEventContent(ctx context.Context) {
	evt, _ := ctx.Value(ctxKeyEvent).(*Event)
	if evt != nil {
		evt.AbortEarly = true
	}
}

func NewBrain() *Brain {
	b := &Brain{
		eventsInput:    make(chan Event),
		eventsLoop:     make(chan Event),
		shutdown:       make(chan shutdownRequest),
		handlers:       make(map[reflect.Type][]eventHandler),
		handlerTimeout: time.Minute,
	}

	b.consumeEvents()

	return b
}

func (b *Brain) isHandlingEvents() bool {
	return atomic.LoadInt32(&b.handlingEvents) == 1
}

func (b *Brain) isClosed() bool {
	return atomic.LoadInt32(&b.closed) == 1
}

func (b *Brain) consumeEvents() {
	var queue []Event

	outChan := func() chan Event {
		if len(queue) == 0 {
			return nil
		}
		return b.eventsLoop
	}

	nextEvt := func() Event {
		if len(queue) == 0 {
			// Prevent index out of bounds if there is no next event. Note that
			// this event is actually never received because the outChan()
			// function above will return "nil" in this case which disables the
			// corresponding select case.
			return Event{}
		}

		return queue[0]
	}

	go func() {
		for {
			select {
			case event, ok := <-b.eventsInput:
				if !ok {
					for _, event := range queue {
						b.eventsLoop <- event
					}
					close(b.eventsLoop)
					return
				}
				queue = append(queue, event)
			case outChan() <- nextEvt():
				queue = queue[1:]
			}
		}
	}()
}

func (b *Brain) RegisterHandler(fun interface{}) {
	err := b.registerHandler(fun)
	if err != nil {
		b.RegistrationErrs = append(b.RegistrationErrs, err)
	}
}

func (b *Brain) registerHandler(fun interface{}) error {
	handler := reflect.ValueOf(fun)
	handlerType := handler.Type()
	if handlerType.Kind() != reflect.Func {
		return errors.New("Event handler is not a function")
	}
	fmt.Println("handlerType", handlerType)

	eventType, withContext, err := checkHandlerParams(handlerType)
	fmt.Println("EventType", eventType)
	if err != nil {
		return err
	}
	returnsErr, err := checkHandlerReturnValues(handlerType)
	if err != nil {
		return err
	}
	handlerFun := newHandlerFunc(handler, withContext, returnsErr)

	b.mu.Lock()
	b.handlers[eventType] = append(b.handlers[eventType], handlerFun)
	b.mu.Unlock()

	return nil
}

func (b *Brain) Emit(event interface{}, callbacks ...func(Event)) {
	b.eventsInput <- Event{Data: event, Callbacks: callbacks}
}

func checkHandlerParams(handlerFunc reflect.Type) (eventType reflect.Type, withContext bool, err error) {
	numParams := handlerFunc.NumIn()
	if numParams == 0 || numParams > 2 {
		err = errors.New("event handler needs one or two arguments")
		return
	}
	eventType = handlerFunc.In(numParams - 1) // eventtype is the last argument
	withContext = numParams == 2

	if withContext {
		contextInterface := reflect.TypeOf((*context.Context)(nil)).Elem()
		if handlerFunc.In(1).Implements(contextInterface) {
			err = errors.New("event handler context must be the first argument")
			return
		}
		if !handlerFunc.In(0).Implements(contextInterface) {
			err = errors.New("event handler has two arguments but the first is not a context.Context")
			return
		}
	}

	if eventType.Kind() == reflect.Ptr {
		err = errors.New("event handler argument cannot be a pointer")
		return
	}
	return eventType, withContext, nil
}

func checkHandlerReturnValues(handlerFunc reflect.Type) (returnsError bool, err error) {
	switch handlerFunc.NumOut() {
	case 0:
		return false, nil
	case 1:
		errorInterface := reflect.TypeOf((*error)(nil)).Elem()
		if !handlerFunc.Out(0).Implements(errorInterface) {
			err = errors.New("if the event handler has a return value it must implement the error interface")
			return
		}
		return true, nil
	default:
		return false, fmt.Errorf("event handler has more than one return value")
	}
}

func newHandlerFunc(handler reflect.Value, withContext, returnsErr bool) eventHandler {
	return func(ctx context.Context, event reflect.Value) (handlerErr error) {
		defer func() {
			if err := recover(); err != nil {
				handlerErr = fmt.Errorf("handler panic: %v", err)
			}
		}()

		var args []reflect.Value
		if withContext {
			args = []reflect.Value{
				reflect.ValueOf(ctx),
				event,
			}
		} else {
			args = []reflect.Value{event}
		}

		results := handler.Call(args)
		if returnsErr && !results[0].IsNil() {
			return results[0].Interface().(error)
		}
		return nil
	}
}

func (b *Brain) HandleEvents() {
	if b.isClosed() {
		return
	}

	ctx := context.Background()
	var shutdown shutdownRequest // set when Brain.Shutdown() is called

	atomic.StoreInt32(&b.handlingEvents, 1)
	b.handleEvent(ctx, Event{Data: events.InitEvent{}})

	for {
		select {
		case evt, ok := <-b.eventsLoop:
			if !ok {
				// Brain.consumeEvents() is done processing all remaining events
				// and we can now safely shutdown the event handler, knowing that
				// all pending events have been processed.
				b.handleEvent(ctx, Event{Data: events.ShutdownEvent{}})
				shutdown.callback <- true
				return
			}

			b.handleEvent(ctx, evt)

		case shutdown = <-b.shutdown:
			// The Brain is shutting down. We have to close the input channel so
			// we doe no longer accept new events and only process the remaining
			// pending events. When the goroutine of Brain.consumeEvents() is
			// done it will close the events loop channel and the case above will
			// use the shutdown callback and return from this function.
			ctx = shutdown.ctx
			close(b.eventsInput)
			atomic.StoreInt32(&b.handlingEvents, 0)
		}
	}
}

type ctxKey string

const ctxKeyEvent ctxKey = "event"

func (b *Brain) handleEvent(ctx context.Context, event Event) {
	eventData := reflect.ValueOf(event.Data)
	_type := eventData.Type()
	handlers := b.determineHandlers(_type)

	ctx = context.WithValue(ctx, ctxKeyEvent, &event)

	for _, handler := range handlers {
		err := b.executeEventHandler(ctx, handler, eventData)
		if err != nil {
			fmt.Println("Event handler failed ", err)
		}

		if event.AbortEarly {
			// Abort handler execution early instead of running any more
			// handlers. The event state may have been changed by a handler, e.g.
			// using the FinishEventContent(…) function.
			break
		}
	}

	for _, callback := range event.Callbacks {
		callback(event)
	}
}

func (b *Brain) executeEventHandler(ctx context.Context, handler eventHandler, event reflect.Value) error {
	if b.handlerTimeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, b.handlerTimeout)
		defer cancel()
	}

	done := make(chan error)
	go func() {
		done <- handler(ctx, event)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *Brain) determineHandlers(eventType reflect.Type) []eventHandler {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var handlers []eventHandler
	for handlerType, hh := range b.handlers {
		if handlerType == eventType {
			handlers = append(handlers, hh...)
		}

		if handlerType.Kind() == reflect.Interface && eventType.Implements(handlerType) {
			handlers = append(handlers, hh...)
		}
	}

	return handlers
}
