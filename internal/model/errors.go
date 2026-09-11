package model

import "errors"

var ErrProjectNotFound = errors.New("project not found")
var ErrRunNotFound = errors.New("run not found")
var ErrInvalidArgument = errors.New("invalid argument")
var ErrGeometryFailed = errors.New("geometry calculation failed")
