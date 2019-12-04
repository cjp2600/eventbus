package eventbuss

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/rafaeljesus/rabbus"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type EventBuss struct {
	ctx     context.Context
	rabbit  string
	service string
	verbose bool
	logger  zerolog.Logger
	rabbus  *rabbus.Rabbus

	// Processing / Post-processing methods
	Marshal   func(interface{}) ([]byte, error)
	Unmarshal func([]byte, interface{}) error
}

type Response struct {
	Message rabbus.ConsumerMessage
	Object  interface{}
}

func NewEventBuss(rabbit string, options ...Option) (*EventBuss, error) {
	eb := &EventBuss{
		rabbit: rabbit,
		logger: zerolog.New(os.Stdout).With().Timestamp().Str("service", "eventbuss").Logger(),
	}

	eb.Marshal = func(v interface{}) ([]byte, error) {
		return json.Marshal(v)
	}
	eb.Unmarshal = func(b []byte, v interface{}) error {
		return json.Unmarshal(b, v)
	}

	for _, o := range options {
		if err := o(eb); err != nil {
			return nil, err
		}
	}

	if eb.ctx == nil {
		eb.ctx = context.Background()
	}
	if len(eb.service) == 0 {
		eb.service = "event-buss"
	}

	r, err := eb.Connect()
	if err != nil {
		return nil, err
	}
	eb.rabbus = r
	Register(eb.service)
	return eb, nil
}

func (e *EventBuss) Push(event Event, object interface{}) {
	r, err := e.Connect()
	if err != nil {
		log.Error().Msgf(err.Error())
	}

	timeout := time.After(time.Second * 3)
	defer func(r *rabbus.Rabbus) {
		if err := r.Close(); err != nil {
			e.logger.Printf("failed to close rabbus connection %s", err)
		}
	}(r)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go r.Run(ctx)

	b, err := e.Marshal(&object)
	if err != nil {
		log.Error().Msgf("marshal event=%d failed: %s", event, err)
	}

	config := GetEventConfig(event)
	msg := rabbus.Message{
		Exchange:     config.Exchange,
		Kind:         "direct",
		Key:          config.Key,
		Payload:      b,
		DeliveryMode: rabbus.Persistent,
	}

	r.EmitAsync() <- msg

outer:
	for {
		select {
		case <-r.EmitOk():
			e.logger.Info().Msgf("Message was sent")
			break outer
		case err := <-r.EmitErr():
			e.logger.Error().Msgf("Failed to send Message %s", err)
			break outer
		case <-timeout:
			e.logger.Error().Msgf("got time out error")
			break outer
		}
	}
}

func (e *EventBuss) Listening(event Event, handler func(object []byte) error) {
	defer func(r *rabbus.Rabbus) {
		if err := r.Close(); err != nil {
			e.logger.Printf("failed to close rabbus connection %s", err)
		}
	}(e.rabbus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go e.rabbus.Run(ctx)

	messages, err := e.rabbus.Listen(GetEventConfig(event))
	if err != nil {
		e.logger.Printf("Failed to create listener %s", err)
		return
	}
	defer close(messages)

	for {
		e.logger.Printf("Listening for event %d messages...", event)

		m, ok := <-messages
		if !ok {
			e.logger.Printf("Stop listening messages!")
			break
		}

		m.Ack(false)
		e.logger.Printf("Message was consumed")

		err := handler(m.Body)
		if err != nil {
			e.logger.Error().AnErr("handler error %s", err)
		}
	}
}

func (e *EventBuss) Connect() (*rabbus.Rabbus, error) {
	cbStateChangeFunc := func(name, from, to string) {
		// do something when state is changed
		log.Info().Msgf("cbStateChangeFunc - %s %s %s", name, from, to)
	}
	r, err := rabbus.New(
		e.rabbit,
		rabbus.Durable(true),
		rabbus.Attempts(5),
		rabbus.Sleep(time.Second*2),
		rabbus.Threshold(3),
		rabbus.OnStateChange(cbStateChangeFunc),
	)
	if err != nil {
		e.logger.Printf("Failed to init rabbus connection %s", err)
		return nil, err
	}
	return r, err
}
