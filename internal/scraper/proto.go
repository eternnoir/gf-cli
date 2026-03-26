// Package scraper — proto.go
//
// Encodes flight search parameters as proto3 binary matching the fast-flights
// Info message schema.  We use protowire for manual encoding so we don't need
// a protoc code-generation step in the build.
//
// Proto schema (from https://github.com/AWeirdDev/flights/blob/main/fast_flights/pb/flights.proto):
//
//	message Airport   { string airport = 2; }
//	message FlightData{ string date = 2; Airport from_airport = 13; Airport to_airport = 14; }
//	message Info      { repeated FlightData data = 3; Seat seat = 9;
//	                    repeated Passenger passengers = 8; Trip trip = 19; }
package scraper

import (
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/eternnoir/gf-cli/internal/model"
)

// Seat enum values (mirrors Seat in flights.proto).
const (
	seatEconomy        = 1
	seatPremiumEconomy = 2
	seatBusiness       = 3
	seatFirst          = 4
)

// Trip enum values.
const (
	tripRoundTrip = 1
	tripOneWay    = 2
)

// Passenger enum values.
const (
	passengerAdult = 1
	passengerChild = 2
)

// buildTFS encodes params into the proto3 binary used as the ?tfs= URL value.
func buildTFS(params model.SearchParams) []byte {
	var b []byte

	// field 3: repeated FlightData — outbound leg
	b = protowire.AppendTag(b, 3, protowire.BytesType)
	b = protowire.AppendBytes(b, encodeFlightData(params.Origin, params.Destination, params.Date))

	// field 3: FlightData — return leg (round-trip only)
	if params.ReturnDate != "" {
		b = protowire.AppendTag(b, 3, protowire.BytesType)
		b = protowire.AppendBytes(b, encodeFlightData(params.Destination, params.Origin, params.ReturnDate))
	}

	// field 8: repeated Passenger
	for i := 0; i < params.Adults; i++ {
		b = protowire.AppendTag(b, 8, protowire.VarintType)
		b = protowire.AppendVarint(b, passengerAdult)
	}
	for i := 0; i < params.Children; i++ {
		b = protowire.AppendTag(b, 8, protowire.VarintType)
		b = protowire.AppendVarint(b, passengerChild)
	}

	// field 9: Seat
	b = protowire.AppendTag(b, 9, protowire.VarintType)
	b = protowire.AppendVarint(b, seatFromModel(params.Class))

	// field 19: Trip
	trip := uint64(tripOneWay)
	if params.ReturnDate != "" {
		trip = tripRoundTrip
	}
	b = protowire.AppendTag(b, 19, protowire.VarintType)
	b = protowire.AppendVarint(b, trip)

	return b
}

func encodeFlightData(from, to, date string) []byte {
	var b []byte
	// field 2: date
	b = protowire.AppendTag(b, 2, protowire.BytesType)
	b = protowire.AppendString(b, date)
	// field 13: from_airport
	b = protowire.AppendTag(b, 13, protowire.BytesType)
	b = protowire.AppendBytes(b, encodeAirport(from))
	// field 14: to_airport
	b = protowire.AppendTag(b, 14, protowire.BytesType)
	b = protowire.AppendBytes(b, encodeAirport(to))
	return b
}

func encodeAirport(code string) []byte {
	var b []byte
	// field 2: airport string
	b = protowire.AppendTag(b, 2, protowire.BytesType)
	b = protowire.AppendString(b, code)
	return b
}

func seatFromModel(class model.SeatClass) uint64 {
	switch class {
	case model.SeatClassPremiumEconomy:
		return seatPremiumEconomy
	case model.SeatClassBusiness:
		return seatBusiness
	case model.SeatClassFirst:
		return seatFirst
	default:
		return seatEconomy
	}
}
