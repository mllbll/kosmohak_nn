package model

import (
	"encoding/json"
	"fmt"
)

// Snapshot состояние сети в момент t (выход geometry.py)
type Snapshot struct {
	TS           float64                       `json:"t_s"`
	Satellites   []SatState                    `json:"satellites"`
	Edges        []Edge                        `json:"edges"`
	ElevationDeg map[string]map[string]float64 `json:"elevation_deg"`
}

type SatState struct {
	ID     string  `json:"id"`
	XKm    float64 `json:"x_km"`
	YKm    float64 `json:"y_km"`
	ZKm    float64 `json:"z_km"`
	Active bool    `json:"active"`
}

// Edge двунаправленный контакт [id_1, id_2, distance_km]
type Edge struct {
	A          string
	B          string
	DistanceKM float64
}

func (e *Edge) UnmarshalJSON(b []byte) error {
	var raw []any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if len(raw) != 3 {
		return fmt.Errorf("edge must be [id, id, km], got %d values", len(raw))
	}
	a, ok1 := raw[0].(string)
	c, ok2 := raw[1].(string)
	d, ok3 := raw[2].(float64)
	if !ok1 || !ok2 || !ok3 {
		return fmt.Errorf("invalid edge payload: %v", raw)
	}
	e.A, e.B, e.DistanceKM = a, c, d
	return nil
}

func (e Edge) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{e.A, e.B, e.DistanceKM})
}
