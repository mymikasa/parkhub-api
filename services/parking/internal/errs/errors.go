package errs

import "errors"

var (
	ErrParkingLotNotFound        = errors.New("parking lot not found")
	ErrParkingLotAlreadyExists   = errors.New("parking lot already exists")
	ErrParkingLotInvalidStatus   = errors.New("invalid parking lot status transition")
	ErrParkingLotInvalidCapacity = errors.New("invalid parking lot capacity")
	ErrParkingLotNameDuplicate   = errors.New("parking lot name already exists under this tenant")
	ErrLaneNotFound              = errors.New("lane not found")
	ErrLaneNameDuplicate         = errors.New("lane name already exists in this parking lot")
)
