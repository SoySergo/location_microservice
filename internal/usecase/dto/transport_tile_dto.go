package dto

// TransportTileRequest - запрос на получение транспортного тайла
type TransportTileRequest struct {
	Z     int      `json:"z" validate:"required,min=0,max=18"`
	X     int      `json:"x" validate:"required,min=0"`
	Y     int      `json:"y" validate:"required,min=0"`
	Types []string `json:"types,omitempty" validate:"omitempty,dive,oneof=metro bus tram cercania long_distance"`
}
