package schema

import "github.com/doug-martin/goqu/v9"

const (
	VehicleVehicleTypeTableName                 = "vehicle_vehicle_type"
	VehicleVehicleTypeTableVehicleTypeIDColName = "vehicle_type_id"
	VehicleVehicleTypeTableVehicleIDColName     = "vehicle_id"
	VehicleVehicleTypeTableInheritedColName     = "inherited"
)

var (
	VehicleVehicleTypeTable                 = goqu.T(VehicleVehicleTypeTableName)
	VehicleVehicleTypeTableVehicleTypeIDCol = VehicleVehicleTypeTable.Col(VehicleVehicleTypeTableVehicleTypeIDColName)
	VehicleVehicleTypeTableVehicleIDCol     = VehicleVehicleTypeTable.Col(VehicleVehicleTypeTableVehicleIDColName)
	VehicleVehicleTypeTableInheritedCol     = VehicleVehicleTypeTable.Col(VehicleVehicleTypeTableInheritedColName)
)
