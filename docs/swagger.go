// Package docs Location Microservice API.
//
// Микросервис для работы с геопространственными данными из OpenStreetMap.
// Предоставляет API для поиска административных границ, транспортной инфраструктуры,
// точек интереса (POI), а также векторных тайлов для визуализации на картах.
//
// Основные возможности:
// - Поиск и обратное геокодирование административных границ
// - Поиск ближайших транспортных станций (метро, автобусы, трамваи)
// - Поиск точек интереса по категориям в радиусе
// - Получение векторных тайлов (MVT/PBF) для всех типов данных
// - Статистика по загруженным данным
//
//	Schemes: http, https
//	BasePath: /
//	Version: 1.0.0
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//	- application/x-protobuf
//	- application/vnd.mapbox-vector-tile
//
//	Security:
//	- api_key:
//
//	SecurityDefinitions:
//	api_key:
//	     type: apiKey
//	     name: Authorization
//	     in: header
//
// swagger:meta
package docs
