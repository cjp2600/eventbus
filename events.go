package eventbuss

import (
	"github.com/rafaeljesus/rabbus"
)

type Event int

const (
	UserRegistration  Event = 1
	UserAuthorization Event = 2
)

var Configs = make(map[Event]rabbus.ListenConfig)

func Register(queue string) {
	Configs[UserRegistration] = rabbus.ListenConfig{
		Exchange: "event_buss_exch_user_register",
		Kind:     "direct",
		Key:      "event_buss_key_user_register",
		Queue:    queue,
	}
	Configs[UserAuthorization] = rabbus.ListenConfig{
		Exchange: "event_buss_exch_user_authorization",
		Kind:     "direct",
		Key:      "event_buss_key_user_authorization",
		Queue:    queue,
	}
}

func GetEventConfig(event Event) rabbus.ListenConfig {
	if v, ok := Configs[event]; ok {
		return v
	}
	return rabbus.ListenConfig{}
}
